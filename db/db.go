package db

import (
	"database/sql"
	"fmt"
)

type DB struct {
	DB *sql.DB
}

func New(db *sql.DB) DB {
	return DB{db}
}

func (db *DB) CountTable(table string) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s;", table)

	row, err := db.DB.Query(query)
	if err != nil {
		return 0, err
	}
	defer row.Close()

	row.Next()

	var res int64
	return res, row.Scan(&res)
}
