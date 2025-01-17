package main

import (
	"database/sql"

	_ "github.com/lib/pq"
)

func GetDatabase(address string) (*sql.DB, error) {
	db, err := sql.Open("postgres", address)
	if err != nil {
		return nil, err
	}

	return db, err
}
