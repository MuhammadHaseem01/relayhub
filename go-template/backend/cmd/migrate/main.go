package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	databaseURL := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/nextroutex?sslmode=disable")
	migrationsDir := env("MIGRATIONS_DIR", "database/migrations")

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}
	if err := ensureMigrationsTable(ctx, db); err != nil {
		log.Fatal(err)
	}

	files, err := migrationFiles(migrationsDir)
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		if err := applyMigration(ctx, db, file); err != nil {
			log.Fatal(err)
		}
	}
}

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS "schema_migrations" (
		"version" TEXT PRIMARY KEY,
		"appliedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func migrationFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	files := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func applyMigration(ctx context.Context, db *sql.DB, file string) error {
	version := filepath.Base(file)
	var exists bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM "schema_migrations" WHERE "version" = $1)`, version).Scan(&exists); err != nil {
		return err
	}
	if exists {
		fmt.Printf("skip %s\n", version)
		return nil
	}

	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, string(raw)); err != nil {
		return fmt.Errorf("%s: %w", version, err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO "schema_migrations" ("version") VALUES ($1)`, version); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	fmt.Printf("applied %s\n", version)
	return nil
}

func env(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
