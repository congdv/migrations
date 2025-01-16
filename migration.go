package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

var QueryTimeoutDuration = 5 * time.Second

type Migration struct {
	db       *sql.DB
	tx       *sql.Tx
	versions []MigrationVersion
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

func withTx(db *sql.DB, ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		fmt.Printf("[Migrations] Rolling back migrations...\n")
		return err
	}

	return tx.Commit()
}

func (m *MigrationVersion) Forward(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, m.UpQuery)
	return err
}
func (m *MigrationVersion) Backward(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, m.DownQuery)
	return err
}

func (m *Migration) InitializeMigrationTable(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS migrations (
		id          bigserial PRIMARY KEY,
		name        TEXT UNIQUE NOT NULL,
		created_at  timestamp(0) with time zone NOT NULL DEFAULT NOW()
	)
`
	_, err := m.tx.ExecContext(ctx, query)
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
	_, err := m.tx.ExecContext(ctx, query)
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

	rows, err := m.tx.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(existingMigrations) == 0 {
		return nil, nil
	}

	return existingMigrations, nil
}

func (m *Migration) addMigrationVersion(ctx context.Context, migrationName string) error {
	query := `
		INSERT INTO migrations(name)
		VALUES ($1)
	`

	_, err := m.tx.ExecContext(ctx, query, migrationName)
	return err
}

func (m *Migration) removeMigrationVersion(ctx context.Context, migrationName string) error {
	query := `
		DELETE FROM migrations
		WHERE name = $1
	`

	_, err := m.tx.ExecContext(ctx, query, migrationName)
	return err
}

func (migration *Migration) MigrateUp(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()

	return withTx(migration.db, ctx, func(tx *sql.Tx) error {
		fmt.Printf("[Migrations] Running up database migrations\n")
		migration.tx = tx
		migration.InitializeMigrationTable(ctx)
		existingMigrations, err := migration.getExitingMigrations(ctx)
		if err != nil {
			return err
		}
		for _, m := range migration.versions {
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

			err := m.Forward(ctx, migration.tx)
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
	})

}

func (migration *Migration) MigrateDown(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeoutDuration)
	defer cancel()
	return withTx(migration.db, ctx, func(tx *sql.Tx) error {
		fmt.Printf("[Migrations] Running down database migrations\n")
		migration.tx = tx
		existingMigrations, err := migration.getExitingMigrations(ctx)
		if err != nil {
			return err
		}
		for _, m := range migration.versions {
			hasSkipped := true
			for _, r := range existingMigrations {
				if r.Name == m.Name {
					if err := m.Backward(ctx, migration.tx); err != nil {
						return err
					}

					if err := migration.removeMigrationVersion(ctx, m.Name); err != nil {
						return err
					}
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
		fmt.Printf("Completed destroy migrations\n")
		return nil
	})

}
