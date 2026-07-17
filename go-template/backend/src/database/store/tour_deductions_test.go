package store

import (
	"testing"
	"time"
)

func TestTourIDFromText(t *testing.T) {
	tests := map[string]int{
		"TOUR-1234567890-17": 17,
		"17":                 17,
		"bad-value":          0,
		"":                   0,
	}
	for input, want := range tests {
		if got := TourIDFromText(input); got != want {
			t.Fatalf("TourIDFromText(%q) = %d, want %d", input, got, want)
		}
	}
}

func TestTourStatusFromDeduction(t *testing.T) {
	if got := TourStatusFromDeduction("Late Delivery"); got != "Late Delivery" {
		t.Fatalf("Late Delivery mapped to %q", got)
	}
	if got := TourStatusFromDeduction("Late_Delivery"); got != "Late Delivery" {
		t.Fatalf("Late_Delivery mapped to %q", got)
	}
	if got := TourStatusFromDeduction("Cancelled"); got != "Cancelled" {
		t.Fatalf("Cancelled mapped to %q", got)
	}
}

func TestNormalizeValueJSONStringColumn(t *testing.T) {
	got := normalizeValue(map[string]any{"id": "1", "name": "Client"}, Column{JSONString: true})
	if got != `{"id":"1","name":"Client"}` {
		t.Fatalf("normalizeValue() = %#v, want JSON object string", got)
	}
}

func TestValueMissingForCreateDefaultWithEmptyString(t *testing.T) {
	if !valueMissingForCreateDefault("", Column{Default: "Pending"}) {
		t.Fatalf("expected empty string to use default")
	}
}

func TestNormalizeValueStringColumn(t *testing.T) {
	if got := normalizeValue(12345, Column{String: true}); got != "12345" {
		t.Fatalf("normalizeValue() = %#v, want %q", got, "12345")
	}
}

func TestNormalizeValueDateTimeYearMonth(t *testing.T) {
	got, ok := normalizeValue("2026-12", Column{DateTime: true}).(time.Time)
	if !ok {
		t.Fatalf("normalizeValue() did not return time.Time")
	}
	if got.Year() != 2026 || got.Month() != time.December || got.Day() != 1 {
		t.Fatalf("normalizeValue() = %s, want 2026-12-01", got.Format(time.RFC3339))
	}
}

func TestNamedValueFromStoredIDOnlyJSON(t *testing.T) {
	got := namedValueFromStored(`{"id":"2"}`)
	if got.ID != "2" || got.Name != "" {
		t.Fatalf("namedValueFromStored() = %#v, want id 2 with empty name", got)
	}
}
