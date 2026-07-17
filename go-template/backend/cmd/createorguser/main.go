package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	databaseURL := env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/nextroutex?sslmode=disable")
	organizationName := env("ORG_NAME", "Cargonex")
	organizationID := env("ORG_ID", createOrganizationID())
	name := env("USER_NAME", "Organization Admin")
	email := strings.ToLower(env("USER_EMAIL", "admin@example.com"))
	password := env("USER_PASSWORD", "password123")
	role := env("USER_ROLE", "Admin")
	status := env("USER_STATUS", "Active")
	permissions := permissionsFromEnv()

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatal(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		log.Fatal(err)
	}

	var userID int
	err = db.QueryRowContext(ctx, `INSERT INTO "User" (
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
	ON CONFLICT ("email") DO UPDATE SET
		"name" = EXCLUDED."name",
		"password" = EXCLUDED."password",
		"organizationName" = EXCLUDED."organizationName",
		"organizationId" = EXCLUDED."organizationId",
		"role" = EXCLUDED."role",
		"permissions" = EXCLUDED."permissions",
		"status" = EXCLUDED."status",
		"updatedAt" = NOW()
	RETURNING "id"`,
		name,
		email,
		string(hash),
		organizationName,
		organizationID,
		role,
		pq.Array(permissions),
		status,
	).Scan(&userID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("organization: %s (%s)\n", organizationName, organizationID)
	fmt.Printf("user: %s <%s> id=%d role=%s status=%s permissions=%s\n", name, email, userID, role, status, strings.Join(permissions, ","))
}

func permissionsFromEnv() []string {
	raw := strings.TrimSpace(os.Getenv("USER_PERMISSIONS"))
	if raw == "" || strings.EqualFold(raw, "all") {
		if permissions := permissionsFromEnumFile(); len(permissions) > 0 {
			return permissions
		}
		return []string{"*"}
	}

	parts := strings.Split(raw, ",")
	permissions := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			permissions = append(permissions, part)
		}
	}
	if len(permissions) == 0 {
		return []string{"*"}
	}
	return permissions
}

func permissionsFromEnumFile() []string {
	paths := []string{
		env("PERMISSIONS_FILE", ""),
		"../src/modules/user/enums/permissionEnum.ts",
		"src/modules/user/enums/permissionEnum.ts",
	}
	seen := map[string]bool{}
	permissions := []string{}
	valueRe := regexp.MustCompile(`=\s*["']([^"']+)["']`)

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		matches := valueRe.FindAllStringSubmatch(string(raw), -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			value := strings.TrimSpace(match[1])
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			permissions = append(permissions, value)
		}
		if len(permissions) > 0 {
			return permissions
		}
	}

	return permissions
}

func createOrganizationID() string {
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	return "ORG-" + strings.ToUpper(hex.EncodeToString(buf))
}

func env(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
