package main

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
)

// Schema changes live in migrations/ as numbered SQL files (001_initial.sql,
// 002_something.sql, ...). On startup the server applies, in order, every file
// it hasn't applied yet, each in its own transaction. Never edit a file that has
// shipped: add a new one instead.
//
//go:embed migrations/*.sql
var migrations embed.FS

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    integer PRIMARY KEY,
		applied_at timestamptz NOT NULL DEFAULT now()
	)`); err != nil {
		return err
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	for _, f := range files {
		version, err := strconv.Atoi(strings.SplitN(f.Name(), "_", 2)[0])
		if err != nil {
			return fmt.Errorf("migration %s: name must start with a number", f.Name())
		}
		body, err := migrations.ReadFile("migrations/" + f.Name())
		if err != nil {
			return err
		}
		if err := applyMigration(db, version, string(body)); err != nil {
			return fmt.Errorf("migration %s: %w", f.Name(), err)
		}
	}
	return nil
}

func applyMigration(db *sql.DB, version int, body string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Holding this lock until commit means two servers starting at once take
	// turns instead of both applying the same migration.
	if _, err := tx.Exec(`SELECT pg_advisory_xact_lock(7343001)`); err != nil {
		return err
	}
	var done bool
	if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&done); err != nil || done {
		return err
	}
	if _, err := tx.Exec(body); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
		return err
	}
	log.Printf("applied database migration %d", version)
	return tx.Commit()
}
