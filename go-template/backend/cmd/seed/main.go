package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strings"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	databaseURL := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/nextroutex?sslmode=disable")
	email := strings.ToLower(env("SEED_ADMIN_EMAIL", "admin@example.com"))
	password := env("SEED_ADMIN_PASSWORD", "password123")
	name := env("SEED_ADMIN_NAME", "Seed Admin")
	organizationName := env("SEED_ORGANIZATION_NAME", "Cargonex")
	organizationID := env("SEED_ORGANIZATION_ID", "ORG-SEED")

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.ExecContext(ctx, `INSERT INTO "User" (
		"name",
		"email",
		"password",
		"organizationName",
		"organizationId",
		"role",
		"permissions",
		"status",
		"createdById",
		"updatedAt"
	) VALUES ($1, $2, $3, $4, $5, $6::"Role", $7, $8::"Status", NULL, NOW())
	ON CONFLICT ("email") DO NOTHING`,
		name,
		email,
		string(hash),
		organizationName,
		organizationID,
		"Super Admin",
		pq.Array([]string{"*"}),
		"Active",
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("seed admin ready: %s", email)
}

func env(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
