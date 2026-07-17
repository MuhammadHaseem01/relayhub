package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type LoginResult struct {
	ID int
}

func (s *Store) LoginUser(ctx context.Context, email string, password string) (LoginResult, error) {
	var id int
	var hash string
	var status string
	err := s.db.QueryRowContext(ctx, `SELECT "id", "password", "status"::TEXT FROM "User" WHERE "email" = $1`, strings.ToLower(email)).Scan(&id, &hash, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return LoginResult{}, ErrNotFound
	}
	if err != nil {
		return LoginResult{}, err
	}
	if status != "Active" {
		return LoginResult{}, ErrForbidden
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return LoginResult{}, ErrUnauthorized
	}
	return LoginResult{ID: id}, nil
}

func (s *Store) RegisterUser(ctx context.Context, body map[string]any, currentUserID int) error {
	email := lowerString(body["email"])
	password, _ := body["password"].(string)
	name, _ := body["name"].(string)
	role, _ := body["role"].(string)
	status, _ := body["status"].(string)
	permissions := stringSlice(body["permissions"])
	if email == "" || password == "" || name == "" || role == "" || status == "" {
		return ErrBadRequest
	}
	var existing int
	err := s.db.QueryRowContext(ctx, `SELECT "id" FROM "User" WHERE "email" = $1`, email).Scan(&existing)
	if err == nil {
		return ErrConflict
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	organizationName, organizationID, err := s.resolveCreationOrganization(ctx, body, currentUserID)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}
	if role == "Manager" || role == "Super Admin" {
		_, _ = s.db.ExecContext(ctx, fmt.Sprintf(`ALTER TYPE "Role" ADD VALUE IF NOT EXISTS '%s'`, strings.ReplaceAll(role, "'", "''")))
	}
	var newID int
	err = s.db.QueryRowContext(ctx, `INSERT INTO "User" ("name", "email", "password", "organizationName", "organizationId", "role", "permissions", "status", "createdById", "updatedAt") VALUES ($1, $2, $3, $4, $5, $6::"Role", $7, $8::"Status", $9, $10) RETURNING "id"`,
		name, email, string(hash), organizationName, organizationID, role, pq.Array(permissions), status, currentUserID, time.Now().UTC()).Scan(&newID)
	if strings.Contains(fmt.Sprint(err), "duplicate key") {
		return ErrConflict
	}
	return err
}

func (s *Store) resolveCreationOrganization(ctx context.Context, body map[string]any, currentUserID int) (string, string, error) {
	var role, organizationName string
	var organizationID sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT "role"::TEXT, "organizationName", "organizationId" FROM "User" WHERE "id" = $1`, currentUserID).Scan(&role, &organizationName, &organizationID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", ErrUnauthorized
	}
	if err != nil {
		return "", "", err
	}
	bodyOrganization, _ := body["organizationName"].(string)
	if role == "Admin" {
		orgID := organizationID.String
		if orgID == "" {
			orgID = createOrganizationID()
			_, err = s.db.ExecContext(ctx, `UPDATE "User" SET "organizationId" = $1 WHERE "id" = $2`, orgID, currentUserID)
		}
		return organizationName, orgID, err
	}
	if role == "Super Admin" {
		if bodyOrganization == "" {
			return "", "", ErrBadRequest
		}
		return bodyOrganization, createOrganizationID(), nil
	}
	if bodyOrganization == "" {
		return "", "", ErrBadRequest
	}
	orgID := organizationID.String
	if orgID == "" {
		orgID = createOrganizationID()
		_, err = s.db.ExecContext(ctx, `UPDATE "User" SET "organizationId" = $1 WHERE "id" = $2`, orgID, currentUserID)
	}
	return bodyOrganization, orgID, err
}

func (s *Store) ListUsers(ctx context.Context, currentUserID int, page int, limit int) ([]map[string]any, Meta, error) {
	cfg := CRUDConfig{Table: `"User"`, SelectColumns: userSelectColumns()}
	rows, meta, err := s.ListOwned(ctx, cfg, currentUserID, page, limit)
	if err != nil {
		return nil, Meta{}, err
	}
	for _, row := range rows {
		normalizeUserRow(row)
	}
	return rows, meta, nil
}

func (s *Store) GetAccessibleUser(ctx context.Context, id int, currentUserID int) (map[string]any, error) {
	canAll, err := s.canAccessAllUsers(ctx, currentUserID)
	if err != nil {
		return nil, err
	}
	query := `SELECT ` + quotedColumns(userSelectColumns()) + ` FROM "User" WHERE "id" = $1`
	args := []any{id}
	if !canAll {
		query += ` AND ("id" = $2 OR "createdById" = $2)`
		args = append(args, currentUserID)
	}
	query += ` LIMIT 1`
	rows, err := s.queryMaps(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	normalizeUserRow(rows[0])
	return rows[0], nil
}

func (s *Store) UpdateUser(ctx context.Context, id int, currentUserID int, body map[string]any) (map[string]any, error) {
	values := map[string]any{}
	casts := map[string]string{}
	if v, ok := body["name"]; ok {
		values["name"] = v
	}
	if v, ok := body["email"]; ok {
		values["email"] = lowerString(v)
	}
	if v, ok := body["password"].(string); ok {
		hash, err := bcrypt.GenerateFromPassword([]byte(v), 12)
		if err != nil {
			return nil, err
		}
		values["password"] = string(hash)
	}
	if v, ok := body["organizationName"]; ok {
		values["organizationName"] = v
	}
	if v, ok := body["role"]; ok {
		values["role"] = v
		casts["role"] = "Role"
	}
	if v, ok := body["permissions"]; ok {
		values["permissions"] = pq.Array(stringSlice(v))
	}
	if v, ok := body["status"]; ok {
		values["status"] = v
		casts["status"] = "Status"
	}
	if len(values) == 0 {
		return nil, ErrEmptyUpdate
	}
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT "id" FROM "User" WHERE "id" = $1`, id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return map[string]any{"message": "User updated successfully"}, nil
	}
	if err != nil {
		return nil, err
	}
	if _, err := s.GetAccessibleUser(ctx, id, currentUserID); err != nil {
		return nil, err
	}
	values["updatedAt"] = time.Now().UTC()
	row, err := s.updateReturning(ctx, `"User"`, userSelectColumns(), id, values, casts)
	if strings.Contains(fmt.Sprint(err), "duplicate key") {
		return nil, ErrConflict
	}
	if err != nil {
		return nil, err
	}
	normalizeUserRow(row)
	return row, nil
}

func (s *Store) DeleteUser(ctx context.Context, id int, currentUserID int) error {
	if _, err := s.GetAccessibleUser(ctx, id, currentUserID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, table := range []string{"User", "Client", "Driver", "Vehicle", "VehicleMaintenance", "TourDeduction", "Tour", "TourDamage"} {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE "%s" SET "createdById" = NULL WHERE "createdById" = $1`, table), id); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM "User" WHERE "id" = $1`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) canAccessAllUsers(ctx context.Context, currentUserID int) (bool, error) {
	var role string
	err := s.db.QueryRowContext(ctx, `SELECT "role"::TEXT FROM "User" WHERE "id" = $1`, currentUserID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrUnauthorized
	}
	if err != nil {
		return false, err
	}
	return role == "Admin" || role == "Super Admin", nil
}

func userSelectColumns() []string {
	return []string{"id", "name", "email", "organizationName", "role", "permissions", "status", "createdAt", "updatedAt", "createdById"}
}

func normalizeUserRow(row map[string]any) {
	if role, ok := row["role"].(string); ok {
		row["role"] = strings.ReplaceAll(role, "_", " ")
	}
	row["permissions"] = normalizePermissions(row["permissions"])
}

func normalizePermissions(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case string:
		v = strings.Trim(v, "{}")
		if v == "" {
			return []string{}
		}
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.Trim(part, `"`)
			if part != "" {
				out = append(out, part)
			}
		}
		return out
	}
	return []string{}
}

func lowerString(v any) string {
	s, _ := v.(string)
	return strings.ToLower(s)
}

func stringSlice(v any) []string {
	raw, ok := v.([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func createOrganizationID() string {
	buf := make([]byte, 4)
	_, _ = rand.Read(buf)
	return "ORG-" + strings.ToUpper(hex.EncodeToString(buf))
}

func (s *Store) UserOrganization(ctx context.Context, userID int) (string, string, error) {
	var organizationID, organizationName sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT "organizationId", "organizationName" FROM "User" WHERE "id" = $1`, userID).Scan(&organizationID, &organizationName)
	if err != nil {
		return "", "", err
	}
	return organizationID.String, organizationName.String, nil
}
