package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type DBMigration struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

type MigrationVersion struct {
	Name     string
	Forward  func(ctx context.Context, db *sql.DB) error
	Backward func(ctx context.Context, db *sql.DB) error
}

func getExitingMigrations(ctx context.Context, db *sql.DB) ([]DBMigration, error) {
	query := `
		SELECT * FROM migrations ORDER BY name ASC
	`
	existingMigrations := []DBMigration{}

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var m DBMigration
		err := rows.Scan(
			&m.ID,
			&m.Name,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		existingMigrations = append(existingMigrations, m)
	}

	return existingMigrations, nil
}

func MigrateUpDatabase(ctx context.Context, db *sql.DB) error {
	fmt.Printf("[Migrations] Running up database migrations\n")

	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			id          bigserial PRIMARY KEY,
			name        TEXT NOT NULL ,
			created_at  timestamp(0) with time zone NOT NULL DEFAULT NOW()
		)
	`
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	fmt.Printf("[Migrations] Migrations table was created\n")

	existingMigrations, err := getExitingMigrations(ctx, db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		hasMigrated := false
		for _, r := range existingMigrations {
			if r.Name == m.Name {
				hasMigrated = true
				break
			}
		}
		if hasMigrated {
			fmt.Printf("[Migrations] Skipping completed migrations [%s]\n", m.Name)
			continue
		}

		err := m.Forward(ctx, db)
		if err != nil {
			fmt.Printf("[Migrations] Failed to run migration [%s]\n", m.Name)
			return err
		}

		query = `
			INSERT INTO migrations(name)
			VALUES ($1)
		`

		_, err = db.ExecContext(ctx, query, m.Name)
		if err != nil {
			return err
		}
		fmt.Printf("Completed migration for [%s]\n", m.Name)
	}
	return nil
}

func MigrateDownDatabase(ctx context.Context, db *sql.DB) error {
	fmt.Printf("[Migrations] Running down database migrations\n")
	existingMigrations, err := getExitingMigrations(ctx, db)
	if err != nil {
		return err
	}

	for _, m := range migrations {
		hasCleaned := false
		for _, existing := range existingMigrations {
			if existing.Name == m.Name {
				m.Backward(ctx, db)
				hasCleaned = true
			}
		}
		if !hasCleaned {
			fmt.Printf("[Migrations] Rollback of migration [%s] is not found", m.Name)
		} else {
			fmt.Printf("[Migrations] Rollback of migration [%s] is completed", m.Name)
		}

	}

	existingMigrations, err = getExitingMigrations(ctx, db)
	if err != nil {
		return err
	}

	if len(existingMigrations) == 0 {
		query := `
		DROP TABLE IF EXISTS migrations;
	`
		_, err := db.ExecContext(ctx, query)
		if err != nil {
			return err
		}
		fmt.Printf("[Migrations] Rollback migration table")
	}
	return nil
}

var migrations = []MigrationVersion{
	{
		Name: "create_users",
		Forward: func(ctx context.Context, db *sql.DB) error {
			query := `
				CREATE EXTENSION IF NOT EXISTS citext;
				CREATE TABLE IF NOT EXISTS users (
					id          bigserial PRIMARY KEY,
					email       citext UNIQUE NOT NULL,
					username    varchar(255) UNIQUE NOT NULL,
					password    bytea NOT NULL,
					created_at  timestamp(0) with time zone NOT NULL DEFAULT NOW(),
					updated_at  timestamp(0) with time zone NOT NULL DEFAULT NOW()
				);
			`
			_, err := db.ExecContext(ctx, query)
			return err
		},
		Backward: func(ctx context.Context, db *sql.DB) error {
			query := `
				DROP EXTENSION IF EXISTS citext;
				DROP TABLE users IF EXISTS
			`
			_, err := db.ExecContext(ctx, query)
			return err
		},
	},
}
