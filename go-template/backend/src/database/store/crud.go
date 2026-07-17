package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

type Column struct {
	JSON                   string
	DB                     string
	Required               bool
	Lowercase              bool
	NullWhenMissing        bool
	EmptyStringWhenMissing bool
	Cast                   string
	Default                any
	DefaultFromJSON        string
	JSONB                  bool
	JSONString             bool
	EnumValues             []string
	MinZero                bool
	String                 bool
	Number                 bool
	DateTime               bool
	Array                  bool
	MinLength              int
	Email                  bool
}

type Meta struct {
	Page            int  `json:"page"`
	Limit           int  `json:"limit"`
	Total           int  `json:"total"`
	TotalPages      int  `json:"totalPages"`
	HasNextPage     bool `json:"hasNextPage"`
	HasPreviousPage bool `json:"hasPreviousPage"`
}

type CRUDConfig struct {
	Table         string
	SelectColumns []string
	CreateColumns []Column
	UpdateColumns []Column
}

func (s *Store) CreateOwned(ctx context.Context, cfg CRUDConfig, body map[string]any, userID int) (map[string]any, error) {
	values := map[string]any{}
	casts := map[string]string{}
	for _, col := range cfg.CreateColumns {
		v, exists := body[col.JSON]
		if !exists || valueMissingForCreateDefault(v, col) {
			switch {
			case col.DefaultFromJSON != "":
				if defaultValue, ok := body[col.DefaultFromJSON]; ok {
					values[col.DB] = normalizeValue(defaultValue, col)
					casts[col.DB] = col.Cast
					continue
				}
			case col.Default != nil:
				values[col.DB] = normalizeValue(col.Default, col)
				casts[col.DB] = col.Cast
				continue
			case col.Required:
				return nil, ErrBadRequest
			case col.NullWhenMissing:
				values[col.DB] = nil
				casts[col.DB] = col.Cast
				continue
			case col.EmptyStringWhenMissing:
				values[col.DB] = ""
				casts[col.DB] = col.Cast
				continue
			default:
				continue
			}
		}
		values[col.DB] = normalizeValue(v, col)
		casts[col.DB] = col.Cast
	}
	values["createdById"] = userID
	values["updatedAt"] = time.Now().UTC()
	return s.insertReturning(ctx, cfg.Table, cfg.SelectColumns, values, casts)
}

func valueMissingForCreateDefault(v any, col Column) bool {
	if v == nil {
		return true
	}
	s, ok := v.(string)
	if !ok || strings.TrimSpace(s) != "" {
		return false
	}
	return col.DefaultFromJSON != "" || col.Default != nil || col.NullWhenMissing || col.EmptyStringWhenMissing
}

func (s *Store) ListOwned(ctx context.Context, cfg CRUDConfig, userID int, page int, limit int) ([]map[string]any, Meta, error) {
	offset := (page - 1) * limit
	rows, err := s.queryMaps(ctx,
		fmt.Sprintf(`SELECT %s FROM %s WHERE "createdById" = $1 ORDER BY "createdAt" DESC OFFSET $2 LIMIT $3`, quotedColumns(cfg.SelectColumns), cfg.Table),
		userID, offset, limit,
	)
	if err != nil {
		return nil, Meta{}, err
	}
	var total int
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE "createdById" = $1`, cfg.Table), userID).Scan(&total); err != nil {
		return nil, Meta{}, err
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	return rows, Meta{Page: page, Limit: limit, Total: total, TotalPages: totalPages, HasNextPage: page < totalPages, HasPreviousPage: page > 1}, nil
}

func (s *Store) GetOwned(ctx context.Context, cfg CRUDConfig, id int, userID int) (map[string]any, error) {
	rows, err := s.queryMaps(ctx,
		fmt.Sprintf(`SELECT %s FROM %s WHERE "id" = $1 AND "createdById" = $2 LIMIT 1`, quotedColumns(cfg.SelectColumns), cfg.Table),
		id, userID,
	)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0], nil
}

func (s *Store) UpdateOwned(ctx context.Context, cfg CRUDConfig, id int, userID int, body map[string]any) (map[string]any, error) {
	values := map[string]any{}
	casts := map[string]string{}
	for _, col := range cfg.UpdateColumns {
		if v, exists := body[col.JSON]; exists {
			values[col.DB] = normalizeValue(v, col)
			casts[col.DB] = col.Cast
		}
	}
	if len(values) == 0 {
		return nil, ErrEmptyUpdate
	}
	if _, err := s.GetOwned(ctx, cfg, id, userID); err != nil {
		return nil, err
	}
	values["updatedAt"] = time.Now().UTC()
	return s.updateReturning(ctx, cfg.Table, cfg.SelectColumns, id, values, casts)
}

func (s *Store) DeleteOwned(ctx context.Context, cfg CRUDConfig, id int, userID int) error {
	if _, err := s.GetOwned(ctx, cfg, id, userID); err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE "id" = $1`, cfg.Table), id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func normalizeValue(v any, col Column) any {
	if s, ok := v.(string); ok && s == "now()" {
		return time.Now().UTC()
	}
	if col.DateTime {
		if t, ok := parseDateTimeValue(v); ok {
			return t
		}
	}
	if col.JSONString {
		if s, ok := v.(string); ok {
			return s
		}
		raw, _ := json.Marshal(v)
		return string(raw)
	}
	if col.JSONB {
		raw, _ := json.Marshal(v)
		return string(raw)
	}
	if col.String {
		s := fmt.Sprint(v)
		if col.Lowercase {
			return strings.ToLower(s)
		}
		return s
	}
	if s, ok := v.(string); ok && col.Lowercase {
		return strings.ToLower(s)
	}
	return v
}

func parseDateTimeValue(value any) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, true
	case string:
		v = strings.TrimSpace(v)
		if v == "" || v == "Invalid Date" {
			return time.Time{}, false
		}
		if v == "now()" {
			return time.Now().UTC(), true
		}
		for _, layout := range dateTimeLayouts() {
			if t, err := time.Parse(layout, v); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

func dateTimeLayouts() []string {
	return []string{
		time.RFC3339Nano,
		"2006-01",
		"2006-01-02",
		"2006-01-02T15:04",
	}
}

func (s *Store) insertReturning(ctx context.Context, table string, selectColumns []string, values map[string]any, casts map[string]string) (map[string]any, error) {
	keys := sortedKeys(values)
	cols := make([]string, 0, len(keys))
	holders := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys))
	for i, key := range keys {
		cols = append(cols, quoteIdent(key))
		holder := fmt.Sprintf("$%d", i+1)
		if cast := casts[key]; cast != "" {
			holder += `::"` + strings.ReplaceAll(cast, `"`, `""`) + `"`
		}
		holders = append(holders, holder)
		args = append(args, values[key])
	}
	sqlText := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) RETURNING %s`, table, strings.Join(cols, ", "), strings.Join(holders, ", "), quotedColumns(selectColumns))
	rows, err := s.queryMaps(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	return rows[0], nil
}

func (s *Store) updateReturning(ctx context.Context, table string, selectColumns []string, id int, values map[string]any, casts map[string]string) (map[string]any, error) {
	keys := sortedKeys(values)
	assignments := make([]string, 0, len(keys))
	args := make([]any, 0, len(keys)+1)
	for i, key := range keys {
		holder := fmt.Sprintf("$%d", i+1)
		if cast := casts[key]; cast != "" {
			holder += `::"` + strings.ReplaceAll(cast, `"`, `""`) + `"`
		}
		assignments = append(assignments, fmt.Sprintf(`%s = %s`, quoteIdent(key), holder))
		args = append(args, values[key])
	}
	args = append(args, id)
	sqlText := fmt.Sprintf(`UPDATE %s SET %s WHERE "id" = $%d RETURNING %s`, table, strings.Join(assignments, ", "), len(args), quotedColumns(selectColumns))
	rows, err := s.queryMaps(ctx, sqlText, args...)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	return rows[0], nil
}

func (s *Store) queryMaps(ctx context.Context, query string, args ...any) ([]map[string]any, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	out := []map[string]any{}
	for rows.Next() {
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := map[string]any{}
		for i, col := range cols {
			row[col] = scanValue(raw[i])
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func scanValue(v any) any {
	switch x := v.(type) {
	case []byte:
		return string(x)
	case time.Time:
		return x
	case sql.NullString:
		if x.Valid {
			return x.String
		}
		return nil
	default:
		return x
	}
}

func quotedColumns(cols []string) string {
	out := make([]string, len(cols))
	for i, col := range cols {
		out[i] = quoteIdent(col)
	}
	return strings.Join(out, ", ")
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
