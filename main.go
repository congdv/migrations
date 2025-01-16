package main

import (
	"context"
	"log"
	"os"
)

var migrations = []MigrationVersion{
	{
		Name: "create_users",
		UpQuery: `
				CREATE EXTENSION IF NOT EXISTS citext;
				CREATE TABLE IF NOT EXISTS users (
					id          bigserial PRIMARY KEY,
					email       citext UNIQUE NOT NULL,
					username    varchar(255) UNIQUE NOT NULL,
					password    bytea NOT NULL,
					created_at  timestamp(0) with time zone NOT NULL DEFAULT NOW(),
					updated_at  timestamp(0) with time zone NOT NULL DEFAULT NOW()
				);
			`,
		DownQuery: `
				DROP EXTENSION IF EXISTS "citext" CASCADE;
				DROP TABLE IF EXISTS users;
			`,
	},
	{
		Name: "create_users_2",
		UpQuery: `
				CREATE EXTENSION IF NOT EXISTS citext;
				CREATE TABLE IF NOT EXISTS users (
					id          bigserial PRIMARY KEY,
					email       citext UNIQUE NOT NULL,
					username    varchar(255) UNIQUE NOT NULL,
					password    bytea NOT NULL,
					created_at  timestamp(0) with time zone NOT NULL DEFAULT NOW(),
					updated_at  timestamp(0) with time zone NOT NULL DEFAULT NOW()
				);
			`,
		DownQuery: `
				DROP EXTENSION IF EXISTS "citext" CASCADE;
				DROP TABLE IF EXISTS users;
			`,
	},
}

func main() {
	inputs := os.Args[1:]
	db, err := GetDatabase("postgres://postgres:Password@123@localhost/migrations?sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	ctx := context.Background()
	migration := &Migration{db: db, versions: migrations}

	if inputs[0] == "up" {
		err = migration.MigrateUp(ctx)
		if err != nil {
			log.Fatal("Failed to migrate up database: ", err)
		}
	}
	if inputs[0] == "down" {
		err = migration.MigrateDown(ctx)
		if err != nil {
			log.Fatal("Failed to migrate down database: ", err)
		}
	}

}
