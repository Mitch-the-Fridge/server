package main

import (
	"archive/tar"
	"bufio"
	"database/sql"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"server-entry/db"

	"github.com/julienschmidt/httprouter"
	_ "github.com/mattn/go-sqlite3"
)

type QueueItem struct {
	ID        string `json:"id"`
	Directory string `json:"dir"`
}
type EmbeddingQueueItem struct {
	UserID   int64     `json:"user_id"`
	Encoding []float64 `json:"encoding"`
}

var (
	// REVIEW
	faceRecognitionQueue chan QueueItem          = make(chan QueueItem, 5)
	embeddingQueue       chan EmbeddingQueueItem = make(chan EmbeddingQueueItem)

	doneChannel chan string = make(chan string)

	database db.DB

	videoDir string
)

const (
	OUTPUT_DIR = "./videos/"
	DB_PATH    = "./database.db"
)

type FrameInfo struct {
	Millis        int64   `json:"time"`
	FaceLocations [][]int `json:"face_locations"`
	CurrentWeight float64 `json:"current_weight"` // I guess??
}

func (fi FrameInfo) Time() time.Time {
	return time.Unix(0, fi.Millis*int64(time.Millisecond))
}

type BlobInfo struct {
	UUID   string      `json:"uuid"`
	FPS    int         `json:"fps"`
	Frames []FrameInfo `json:"frames"`
}

func rootHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	clipCount, err := database.CountTable("clips")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	accountCount, err := database.CountTable("persons")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	beersInFridge, err := database.CountBeersInFridge()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(200)

	encoder := json.NewEncoder(w)
	encoder.Encode(struct {
		ClipCount     int64 `json:"clip_count"`
		AccountCount  int64 `json:"account_count"`
		BeersInFridge int64 `json:"beers_in_fridge"`
	}{
		ClipCount:     clipCount,
		AccountCount:  accountCount,
		BeersInFridge: beersInFridge,
	})
}

func clipsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	body := tar.NewReader(r.Body)
	defer r.Body.Close()

	extension := ""
	var blobInfo BlobInfo
	tmpDir, err := ioutil.TempDir("/tmp", strconv.FormatInt(time.Now().UnixNano(), 10))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	for {
		header, err := body.Next()
		if err == io.EOF { // all files read
			break
		} else if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if header.FileInfo().IsDir() {
			continue
		}

		ext := filepath.Ext(header.Name)
		switch ext {
		case ".jpg", ".jpeg":
			extension = ext
			file, err := os.Create(filepath.Join(tmpDir, header.Name))
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			defer file.Close()

			if _, err := io.Copy(file, body); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		case ".json":
			decoder := json.NewDecoder(body)
			if err := decoder.Decode(&blobInfo); err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
		}
	}

	output := filepath.Join(videoDir, blobInfo.UUID+".mp4")
	if err := MakeVideo(tmpDir, extension, output, blobInfo.FPS); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	println("done", output)

	if _, err := database.InsertClip(
		db.Clip{
			ID:          blobInfo.UUID,
			FPS:         blobInfo.FPS,
			FrameCount:  int64(len(blobInfo.Frames)),
			Start:       blobInfo.Frames[0].Time(),
			BeginWeight: blobInfo.Frames[0].CurrentWeight,
			EndWeight:   blobInfo.Frames[len(blobInfo.Frames)-1].CurrentWeight,
		},
	); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	faceRecognitionQueue <- QueueItem{
		ID:        blobInfo.UUID,
		Directory: tmpDir,
	}
}

func videoHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fname := filepath.Join(videoDir, ps.ByName("id")+".mp4")

	f, err := os.Open(fname)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer f.Close()

	w.WriteHeader(200)
	if _, err := io.Copy(w, f); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func clipsGetHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	clip, found, err := database.GetClip(ps.ByName("id"))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	} else if !found {
		http.Error(w, "not found", 404)
		return
	}

	w.WriteHeader(200)
	encoder := json.NewEncoder(w)
	encoder.Encode(clip)
}

func meHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user, good := getUserByRequest(w, r)
	if !good {
		return
	}

	embeddings, err := database.GetUserEmbeddings(user.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	user.EmbeddingCount = len(embeddings)

	w.WriteHeader(200)
	encoder := json.NewEncoder(w)
	encoder.Encode(user)
}

func spawnNode() error {
	cmd := exec.Command(
		"nice",
		"-n10",

		"node",
		"index.mjs",
		"../"+DB_PATH,
	)
	workdir, err := os.Getwd()
	if err != nil {
		return err
	}
	cmd.Dir = filepath.Join(workdir, "node")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	go func() {
		for {
			var obj interface{}
			var char byte

			select {
			case embedding := <-embeddingQueue:
				obj = embedding
				char = 'E'
			case item := <-faceRecognitionQueue:
				obj = item
				char = 'W'
			}

			item, err := json.Marshal(obj)
			if err != nil {
				panic(err)
			}

			bytes := []byte{char}
			bytes = append(bytes, item...)
			bytes = append(bytes, '\n')

			stdin.Write(bytes)
		}
	}()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	go func() {
		reader := bufio.NewReader(stdout)
		for {
			id, err := reader.ReadString('\n')
			if err == io.EOF {
				return
			} else if err != nil {
				panic(err)
			}

			doneChannel <- strings.TrimSpace(id)
		}
	}()

	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func transactionLoop() {
	// TODO: actually allow people to grab beers for others.

	for id := range doneChannel {
		log.Printf("received clip %s as being processed from node", id)

		clip, found, err := database.GetClip(id)
		if !found {
			log.Fatalf("clip with id %s not found during transactionLoop!", id)
		} else if err != nil {
			panic(err)
		}

		weightDiff := math.Abs(clip.EndWeight - clip.BeginWeight)
		meta, err := database.GetMeta()
		if err != nil {
			panic(err)
		}
		beerCount := math.Round(weightDiff / float64(meta.BeerWeightGrams))

		grab, found, err := database.GetGrabForClip(id)
		if !found {
			log.Fatalf("grab for clip with id %s not found during transactionLoop!", id)
		} else if err != nil {
			panic(err)
		}

		res, err := database.InsertTransaction(db.Transaction{
			GrabID:     grab.ID,
			GrabbedFor: grab.GrabberGuess,
			Amount:     int64(beerCount),
			Pending:    false,
		})
		if err != nil {
			panic(err)
		}

		transId, err := res.LastInsertId()
		if err != nil {
			panic(err)
		}

		log.Printf("inserted transaction %d for clip %s", transId, id)
	}
}

func main() {
	var err error

	videoDir, err = filepath.Abs(OUTPUT_DIR)
	if err != nil {
		panic(err)
	}

	dbNew := false
	if _, err := os.Stat(DB_PATH); os.IsNotExist(err) {
		dbNew = true
	}

	_db, err := sql.Open("sqlite3", DB_PATH+"?_fk=on")
	if err != nil {
		panic(err)
	}

	if dbNew {
		if err := runSchema(_db); err != nil {
			panic(err)
		}
	}

	database = db.New(_db)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		if err := spawnNode(); err != nil {
			panic(err)
		}
	}()

	go transactionLoop()

	router := httprouter.New()
	router.GET("/", rootHandler)
	router.POST("/clips", clipsHandler)
	router.GET("/video/:id", videoHandler)
	router.GET("/clips/:id", clipsGetHandler)

	router.GET("/me", meHandler)
	router.POST("/me", signupHandler)

	router.POST("/me/embedding", embeddingHandler)

	log.Fatal(http.ListenAndServe(":"+port, router))
}
