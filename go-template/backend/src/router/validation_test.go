package router

import (
	"strings"
	"testing"

	"cargonex-backend/src/database/store"
)

func TestValidateBodyRejectsUnknownFields(t *testing.T) {
	messages := validateBody(map[string]any{"fullName": "A", "extra": true}, []store.Column{{JSON: "fullName", Required: true}}, validationCreate)
	if !containsMessage(messages, "property extra should not exist") {
		t.Fatalf("expected unknown field validation error, got %#v", messages)
	}
}

func TestValidateBodyRequiresFields(t *testing.T) {
	messages := validateBody(map[string]any{}, []store.Column{{JSON: "fullName", Required: true}}, validationCreate)
	if !containsMessage(messages, "fullName should not be empty") {
		t.Fatalf("expected required field validation error, got %#v", messages)
	}
}

func TestValidateBodyAllowsOptionalTourFieldsWithDefaults(t *testing.T) {
	columns := []store.Column{
		{JSON: "startDate", Required: true},
		{JSON: "expectedEndDate", DefaultFromJSON: "startDate"},
		{JSON: "paymentStatus", Cast: "TripPaymentStatus", Default: "Pending"},
		{JSON: "expenses", JSONB: true, Default: []any{}},
		{JSON: "fuelDetails", JSONB: true, Default: []any{}},
	}
	messages := validateBody(map[string]any{"startDate": "2026-06-22"}, columns, validationCreate)
	if len(messages) != 0 {
		t.Fatalf("expected optional tour fields to validate, got %#v", messages)
	}
}

func TestValidateBodyAllowsEmptyOptionalTourFieldsWithDefaults(t *testing.T) {
	columns := []store.Column{
		{JSON: "expectedEndDate", DefaultFromJSON: "startDate"},
		{JSON: "paymentStatus", Cast: "TripPaymentStatus", Default: "Pending"},
		{JSON: "expenses", JSONB: true, Default: []any{}},
		{JSON: "fuelDetails", JSONB: true, Default: []any{}},
	}
	body := map[string]any{"startDate": "2026-06-22", "expectedEndDate": "", "paymentStatus": "", "expenses": "", "fuelDetails": ""}
	messages := validateBody(body, columns, validationCreate)
	if len(messages) != 0 {
		t.Fatalf("expected empty optional tour fields to validate, got %#v", messages)
	}
}

func TestValidateBodyRejectsInvalidDates(t *testing.T) {
	messages := validateBody(map[string]any{"statusDateTime": "Invalid Date"}, []store.Column{{JSON: "statusDateTime", DateTime: true}}, validationCreate)
	if !containsMessage(messages, "statusDateTime must be a valid date") {
		t.Fatalf("expected invalid date validation error, got %#v", messages)
	}
}

func TestValidateBodyAllowsNestCompatibleDateStrings(t *testing.T) {
	for _, value := range []string{
		"2026-12",
		"2026-06-23",
		"2026-06-23T12:30",
		"2026-06-23T12:30:45.000Z",
		"2026-06-23T12:30:45.000+05:00",
	} {
		messages := validateBody(map[string]any{"licenseExpiry": value}, []store.Column{{JSON: "licenseExpiry", DateTime: true}}, validationCreate)
		if len(messages) != 0 {
			t.Fatalf("expected %q to validate, got %#v", value, messages)
		}
	}
}

func TestRequestedStatusNormalizesAvailable(t *testing.T) {
	status, ok := requestedStatus(map[string]any{"status": "Available"})
	if !ok || status != "Avaliable" {
		t.Fatalf("requestedStatus() = %q, %v; want Avaliable, true", status, ok)
	}
}

func TestVehicleStoreConfigNormalizesTextFields(t *testing.T) {
	cfg := vehicleStoreConfig()
	if !cfg.CreateColumns[0].String || !cfg.CreateColumns[1].String {
		t.Fatalf("expected vehicleNumber and mtag create columns to normalize as strings")
	}
}

func TestValidateBodyRejectsEnums(t *testing.T) {
	messages := validateBody(map[string]any{"status": "Bad"}, []store.Column{{JSON: "status", Cast: "VehicleStatus"}}, validationUpdate)
	if len(messages) == 0 || !strings.Contains(messages[0], "status must be one of") {
		t.Fatalf("expected enum validation error, got %#v", messages)
	}
}

func TestValidateBodyRejectsNegativeNumbers(t *testing.T) {
	messages := validateBody(map[string]any{"amount": -1.0}, []store.Column{{JSON: "amount", MinZero: true}}, validationCreate)
	if !containsMessage(messages, "amount must not be less than 0") {
		t.Fatalf("expected numeric min validation error, got %#v", messages)
	}
}

func containsMessage(messages []string, want string) bool {
	for _, message := range messages {
		if message == want {
			return true
		}
	}
	return false
}
