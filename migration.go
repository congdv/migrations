package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Migration struct {
	db *sql.DB
}

type DBMigration struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

type MigrationVersion struct {
	Name      string
	UpQuery   string
	DownQuery string
}

func (m *MigrationVersion) Forward(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, m.UpQuery)
	return err
}
func (m *MigrationVersion) Backward(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, m.DownQuery)
	return err
}

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
				DROP EXTENSION IF EXISTS citext;
				DROP TABLE users IF EXISTS
			`,
	},
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
				DROP EXTENSION IF EXISTS citext;
				DROP TABLE users IF EXISTS
			`,
	},
}

func (m *Migration) InitializeMigrationTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS migrations (
		id          bigserial PRIMARY KEY,
		name        TEXT UNIQUE NOT NULL,
		created_at  timestamp(0) with time zone NOT NULL DEFAULT NOW()
	)
`
	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	fmt.Printf("[Migrations] Migrations table was created\n")
	return nil
}

func (m *Migration) DestroyMigrationTable(ctx context.Context) error {
	query := `
		DROP TABLE IF EXISTS migrations;
	`
	_, err := m.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	fmt.Printf("[Migrations] Migrations table was destroyed\n")
	return nil
}

func (m *Migration) getExitingMigrations(ctx context.Context) ([]DBMigration, error) {
	query := `
		SELECT * FROM migrations ORDER BY name ASC
	`
	existingMigrations := []DBMigration{}

	rows, err := m.db.QueryContext(ctx, query)
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

func (m *Migration) addMigrationVersion(ctx context.Context, migrationName string) error {
	query := `
		INSERT INTO migrations(name)
		VALUES ($1)
	`

	_, err := m.db.ExecContext(ctx, query, migrationName)
	return err
}

func (m *Migration) removeMigrationVersion(ctx context.Context, migrationName string) error {
	query := `
		DELETE FROM migrations
		WHERE name = $1
	`

	_, err := m.db.ExecContext(ctx, query, migrationName)
	return err
}

func (migration *Migration) MigrateUp(ctx context.Context) error {
	fmt.Printf("[Migrations] Running up database migrations\n")
	migration.InitializeMigrationTable(ctx)
	existingMigrations, err := migration.getExitingMigrations(ctx)
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

		err := m.Forward(ctx, migration.db)
		if err != nil {
			fmt.Printf("[Migrations] Failed to run migration [%s]\n", m.Name)
			return err
		}
		err = migration.addMigrationVersion(ctx, m.Name)
		if err != nil {
			return err
		}
		fmt.Printf("Completed migration for [%s]\n", m.Name)
	}
	return nil

}

func (migration *Migration) MigrateDown(ctx context.Context) error {
	fmt.Printf("[Migrations] Running down database migrations\n")
	existingMigrations, err := migration.getExitingMigrations(ctx)
	if err != nil {
		return err
	}
	for _, m := range migrations {
		hasSkipped := true
		for _, r := range existingMigrations {
			if r.Name == m.Name {
				m.Backward(ctx, migration.db)
				migration.removeMigrationVersion(ctx, m.Name)
				fmt.Printf("[Migrations] Cleaned migrations [%s]\n", m.Name)
				hasSkipped = false
				break
			}
		}
		if hasSkipped {
			fmt.Printf("[Migrations] Skipping cleaning migrations [%s]\n", m.Name)
			continue
		}

	}
	existingMigrations, err = migration.getExitingMigrations(ctx)
	if err != nil {
		return err
	}
	if len(existingMigrations) == 0 {
		migration.DestroyMigrationTable(ctx)
	}
	fmt.Printf("Completed cleaning migrations\n")
	return nil
}
