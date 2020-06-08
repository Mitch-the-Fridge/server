package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type SignupRequest struct {
	Name string `json:"name"`
}

func signupHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	var request SignupRequest
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// TODO: use transaction here

	res, err := database.DB.Exec("INSERT INTO persons(name) values(?)", request.Name)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	accountId, err := res.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	sessionId, err := generateSession(accountId)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	cookie := &http.Cookie{
		Name:    "session",
		Value:   sessionId,
		Expires: time.Now().AddDate(1, 0, 0),
	}
	http.SetCookie(w, cookie)

	w.WriteHeader(http.StatusCreated)
}

func embeddingsGetHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	user, good := getUserByRequest(w, r)
	if !good {
		return
	}

	embeddings, err := database.GetUserEmbeddings(user.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(200)
	encoder := json.NewEncoder(w)
	encoder.Encode(embeddings)
}

func embeddingsPostHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	user, good := getUserByRequest(w, r)
	if !good {
		return
	}

	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	r.Body.Close()

	// test if it is a valid embedding
	var embedding []float64
	if err := json.Unmarshal(bytes, &embedding); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if _, err := database.InsertEmbedding(user.ID, bytes); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if _, err := sendToNode(AddEmbeddingRequest{
		UserID:   user.ID,
		Encoding: embedding,
	}); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
