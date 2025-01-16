package main

import (
	"context"
	"log"
	"os"
)

func main() {
	inputs := os.Args[1:]
	db, err := GetDatabase("postgres://postgres:Password@123@localhost/migrations?sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	ctx := context.Background()

	if inputs[0] == "up" {
		err = MigrateUpDatabase(ctx, db)
		if err != nil {
			log.Fatal("Failed to migrate up database: %v", err)
		}
	}
	if inputs[0] == "down" {
		err = MigrateDownDatabase(ctx, db)
		if err != nil {
			log.Fatal("Failed to migrate down database: %v", err)
		}
	}

}
