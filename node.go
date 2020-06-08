package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
)

type QueueableItem interface {
	GetOperation() string
}

type QueueItem struct {
	ID   string        `json:"id"`
	Type string        `json:"type"`
	Item QueueableItem `json:"item"`
}

type AddEmbeddingRequest struct {
	UserID   int64     `json:"user_id"`
	Encoding []float64 `json:"encoding"`
}

func (r AddEmbeddingRequest) GetOperation() string { return "train_embedding" }

type ClipRequest struct {
	ID        string `json:"id"`
	Directory string `json:"dir"`
}

func (r ClipRequest) GetOperation() string { return "clip" }

type Result struct {
	ID     string
	Err    error
	Result []byte
}

var (
	requestResponseMap                = make(map[string]chan Result)
	itemQueue          chan QueueItem = make(chan QueueItem)
)

func sendToNode(item QueueableItem) ([]byte, error) {
	// TODO: uuid is overkill
	uid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}
	id := uid.String()

	ch := make(chan Result)
	requestResponseMap[id] = ch

	itemQueue <- QueueItem{
		ID:   id,
		Type: item.GetOperation(),
		Item: item,
	}

	res := <-ch
	if res.ID != id {
		panic(fmt.Sprintf("%s != %s", res.ID, id))
	}
	return res.Result, res.Err
}

type nodeResponse struct {
	ID     string `json:"id"`
	Result string `json:"result"` // json string
	Error  string `json:"error"`
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
			queueItem := <-itemQueue

			bytes, err := json.Marshal(queueItem)
			if err != nil {
				panic(err)
			}
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
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				return
			} else if err != nil {
				panic(err)
			}

			var output nodeResponse
			if err := json.Unmarshal([]byte(line), &output); err != nil {
				panic(err)
			}

			item, has := requestResponseMap[output.ID]
			if !has {
				log.Printf("[WARN] done with task with id %s but no request response waiter found, ignoring...", output.ID)
				continue
			}

			log.Printf("[INF] done with task with id %s", output.ID)
			item <- Result{
				ID:     output.ID,
				Err:    errors.New(output.Error),
				Result: []byte(output.Result),
			}
			close(item)
		}
	}()

	cmd.Stderr = os.Stderr

	return cmd.Run()
}
