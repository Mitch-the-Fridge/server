package main

import (
	"database/sql"
	"fmt"
)

var schemaEntries = []string{}

func runSchema(db *sql.DB) error {
	panic("TODO")
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, entry := range schemaEntries {
		fmt.Printf("running: %s\n", entry)
		if _, err := tx.Exec(entry); err != nil {
			return err
		}
	}

	return tx.Commit()
}
