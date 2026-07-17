package router

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (s *Server) populateTourRelationNames(ctx context.Context, userID int, body map[string]any) error {
	relations := []struct {
		field string
		table string
		name  string
	}{
		{field: "client", table: `"Client"`, name: `"fullName"`},
		{field: "driver", table: `"Driver"`, name: `"fullName"`},
		{field: "vehicle", table: `"Vehicle"`, name: `"vehicleNumber"`},
	}
	for _, rel := range relations {
		value, ok := body[rel.field]
		if !ok || value == nil {
			continue
		}
		id, name := relationIDName(value)
		if id == "" || (name != "" && name != id) {
			continue
		}
		resolved, err := s.lookupRelationName(ctx, rel.table, rel.name, id, userID)
		if err != nil || resolved == "" {
			continue
		}
		body[rel.field] = map[string]any{"id": id, "name": resolved}
	}
	return nil
}

func relationIDName(value any) (string, string) {
	switch v := value.(type) {
	case map[string]any:
		return relationString(v["id"]), relationString(v["name"])
	case string:
		raw := strings.TrimSpace(v)
		if raw == "" {
			return "", ""
		}
		var obj map[string]any
		if json.Unmarshal([]byte(raw), &obj) == nil {
			return relationString(obj["id"]), relationString(obj["name"])
		}
		return raw, ""
	default:
		return relationString(v), ""
	}
}

func relationString(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func (s *Server) lookupRelationName(ctx context.Context, table string, nameColumn string, id string, userID int) (string, error) {
	var name string
	err := s.db.QueryRowContext(ctx, `SELECT `+nameColumn+` FROM `+table+` WHERE "id" = $1 AND "createdById" = $2 LIMIT 1`, id, userID).Scan(&name)
	return name, err
}
