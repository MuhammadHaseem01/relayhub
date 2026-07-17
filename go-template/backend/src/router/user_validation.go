package router

import (
	"strings"

	"cargonex-backend/src/database/store"
)

func loginColumns() []store.Column {
	return []store.Column{
		{JSON: "email", Required: true, Email: true},
		{JSON: "password", Required: true, MinLength: 8},
	}
}

func userCreateColumns() []store.Column {
	return []store.Column{
		{JSON: "name", Required: true},
		{JSON: "email", Required: true, Email: true},
		{JSON: "password", Required: true, MinLength: 8},
		{JSON: "organizationName"},
		{JSON: "role", Required: true, Cast: "Role"},
		{JSON: "permissions", Required: true, Array: true},
		{JSON: "status", Required: true, Cast: "Status"},
	}
}

func userUpdateColumns() []store.Column {
	return []store.Column{
		{JSON: "name"},
		{JSON: "email", Email: true},
		{JSON: "password", MinLength: 8},
		{JSON: "organizationName"},
		{JSON: "role", Cast: "Role"},
		{JSON: "permissions", Array: true},
		{JSON: "status", Cast: "Status"},
	}
}

func validateLoginBody(body map[string]any) []string {
	messages := []string{}
	for key := range body {
		if key != "email" && key != "password" {
			messages = append(messages, "property "+key+" should not exist")
		}
	}
	email, ok := body["email"].(string)
	if !ok || !strings.Contains(email, "@") || !strings.Contains(email, ".") {
		messages = append(messages, "email must be an email")
	}
	password, ok := body["password"].(string)
	if !ok || len(password) < 8 {
		messages = append(messages, "password must be longer than or equal to 8 characters")
	}
	if !ok {
		messages = append(messages, "password must be a string")
	}
	return messages
}
