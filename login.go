package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
)

type LoginRequest struct {
	Embedding []float64 `json:"embedding"`
}

func (r LoginRequest) GetOperation() string { return "login" }

func loginHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	var loginRequest LoginRequest
	if err := decoder.Decode(&loginRequest); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	res, err := sendToNode(loginRequest)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	var loginResponse struct {
		UserID *int64 `json:"user_id"`
	}

	if err := json.Unmarshal(res, &loginResponse); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	userId := loginResponse.UserID
	if userId == nil {
		http.Error(w, "never seen that face before", 403)
		return
	}

	sessionId, err := generateSession(*userId)
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
