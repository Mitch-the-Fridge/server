package main

import (
	"net/http"
	"server-entry/db"
)

func getUserByRequest(w http.ResponseWriter, r *http.Request) (user db.UserInfo, good bool) {
	cookie, err := r.Cookie("session")
	if err == http.ErrNoCookie {
		http.Error(w, "not logged in", 403)
		return db.UserInfo{}, false
	} else if err != nil {
		http.Error(w, err.Error(), 500)
		return db.UserInfo{}, false
	}

	user, found, err := database.GetUserBySessionId(cookie.Value)
	if !found {
		http.Error(w, "invalid session", 403)
		return db.UserInfo{}, false
	} else if err != nil {
		http.Error(w, err.Error(), 500)
		return db.UserInfo{}, false
	}

	return user, true
}
