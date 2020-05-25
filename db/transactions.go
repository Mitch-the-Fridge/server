package db

import "database/sql"

type Transaction struct {
	ID         int64
	GrabID     int64
	GrabbedFor int64
	Amount     int64
	Pending    bool
}

func (db *DB) InsertTransaction(t Transaction) (sql.Result, error) {
	return db.DB.Exec(
		"INSERT INTO transactions(grab_id, grabbed_for, amount, pending) values(?, ?, ?, ?)",
		t.GrabID,
		t.GrabbedFor,
		t.Amount,
		t.Pending,
	)
}

func (db *DB) CountBeersInFridge() (int64, error) {
	row, err := db.DB.Query("SELECT SUM(amount) FROM transactions;")
	if err != nil {
		return 0, err
	}
	defer row.Close()

	row.Next()

	var res *int64
	if err := row.Scan(&res); err != nil {
		return 0, err
	}

	if res == nil {
		return 0, nil
	} else {
		return *res, nil
	}
}
