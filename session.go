package main

import "github.com/google/uuid"

func generateSession(userId int64) (string, error) {
	sessionUUID, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	sessionId := sessionUUID.String()

	_, err = database.DB.Exec("INSERT INTO login_sessions(id, person_id) values(?, ?)", sessionId, userId)
	if err != nil {
		return "", err
	}

	return sessionId, nil
}
