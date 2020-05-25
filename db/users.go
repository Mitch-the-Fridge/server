package db

import (
	"database/sql"
	"encoding/json"
)

type UserInfo struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	SignupDate string `json:"signup_date"`

	EmbeddingCount int `json:"embedding_count"`
}

func scanUser(s interface {
	Scan(dest ...interface{}) error
}) (UserInfo, error) {
	var u UserInfo
	return u, s.Scan(&u.ID, &u.Name, &u.SignupDate)
}

func (db *DB) GetUserBySessionId(sessionId string) (user UserInfo, found bool, err error) {
	rows, err := db.DB.Query(
		"SELECT * FROM persons WHERE id = (SELECT person_id FROM login_sessions WHERE id = ?);",
		sessionId,
	)
	if err != nil {
		return UserInfo{}, false, err
	}
	defer rows.Close()

	if rows.Next() {
		user, err := scanUser(rows)
		return user, true, err
	} else {
		return UserInfo{}, false, nil
	}
}

func (db *DB) GetUserEmbeddings(userId int64) ([][]float64, error) {
	rows, err := db.DB.Query("SELECT embedding FROM embeddings WHERE user_id = ?", userId)
	if err != nil {
		return [][]float64{}, err
	}
	defer rows.Close()

	var res [][]float64
	for rows.Next() {
		var embeddingRaw []byte
		if err := rows.Scan(&embeddingRaw); err != nil {
			return res, err
		}

		var embedding []float64
		if err := json.Unmarshal(embeddingRaw, &embedding); err != nil {
			return res, err
		}
		res = append(res, embedding)
	}
	return res, nil
}

func (db *DB) InsertEmbedding(userId int64, embeddingBytes []byte) (sql.Result, error) {
	return db.DB.Exec(
		"INSERT INTO embeddings(user_id, embedding) values(?, ?)",
		userId,
		embeddingBytes,
	)
}
