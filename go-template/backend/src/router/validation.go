package router

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"cargonex-backend/src/database/store"
)

type validationMode string

const (
	validationCreate validationMode = "create"
	validationUpdate validationMode = "update"
)

func validateBody(body map[string]any, columns []store.Column, mode validationMode, extraAllowed ...string) []string {
	allowed := map[string]store.Column{}
	for _, col := range columns {
		allowed[col.JSON] = col
		if col.DefaultFromJSON != "" {
			allowed[col.DefaultFromJSON] = store.Column{JSON: col.DefaultFromJSON}
		}
	}
	for _, key := range extraAllowed {
		allowed[key] = store.Column{JSON: key}
	}
	messages := []string{}
	for key := range body {
		if _, ok := allowed[key]; !ok {
			messages = append(messages, fmt.Sprintf("property %s should not exist", key))
		}
	}
	for _, col := range columns {
		value, exists := body[col.JSON]
		if !exists && col.DefaultFromJSON != "" {
			value, exists = body[col.DefaultFromJSON]
		}
		if !exists {
			if mode == validationCreate && col.Required && col.Default == nil && col.DefaultFromJSON == "" {
				messages = append(messages, fmt.Sprintf("%s should not be empty", col.JSON))
			}
			continue
		}
		if valueMissingForValidationDefault(value, col) {
			continue
		}
		messages = append(messages, validateValue(col, value)...)
	}
	return messages
}

func valueMissingForValidationDefault(value any, col store.Column) bool {
	if value == nil {
		return col.DefaultFromJSON != "" || col.Default != nil || col.NullWhenMissing || col.EmptyStringWhenMissing
	}
	s, ok := value.(string)
	if !ok || strings.TrimSpace(s) != "" {
		return false
	}
	return col.DefaultFromJSON != "" || col.Default != nil || col.NullWhenMissing || col.EmptyStringWhenMissing
}

func validateValue(col store.Column, value any) []string {
	messages := []string{}
	if col.Required || col.String {
		if s, ok := value.(string); ok && strings.TrimSpace(s) == "" {
			messages = append(messages, fmt.Sprintf("%s should not be empty", col.JSON))
		}
	}
	if col.Number || col.MinZero || numericField(col.JSON) {
		number, ok := numericValue(value)
		if !ok {
			messages = append(messages, fmt.Sprintf("%s must be a number conforming to the specified constraints", col.JSON))
		} else if col.MinZero && number < 0 {
			messages = append(messages, fmt.Sprintf("%s must not be less than 0", col.JSON))
		}
	}
	if col.Array {
		if _, ok := value.([]any); !ok {
			messages = append(messages, fmt.Sprintf("%s must be an array", col.JSON))
		}
	}
	if col.MinLength > 0 {
		s, ok := value.(string)
		if !ok || len(s) < col.MinLength {
			messages = append(messages, fmt.Sprintf("%s must be longer than or equal to %d characters", col.JSON, col.MinLength))
		}
	}
	if col.Email {
		s, ok := value.(string)
		if !ok || !strings.Contains(s, "@") || !strings.Contains(s, ".") {
			messages = append(messages, fmt.Sprintf("%s must be an email", col.JSON))
		}
	}
	if col.DateTime && !validDateTime(value) {
		messages = append(messages, fmt.Sprintf("%s must be a valid date", col.JSON))
	}
	enumValues := col.EnumValues
	if len(enumValues) == 0 && col.Cast != "" {
		enumValues = enumValuesForCast(col.Cast)
	}
	if len(enumValues) > 0 {
		s, ok := value.(string)
		if !ok || !oneOf(s, enumValues) {
			messages = append(messages, fmt.Sprintf("%s must be one of the following values: %s", col.JSON, strings.Join(enumValues, ", ")))
		}
	}
	return messages
}

func validDateTime(value any) bool {
	switch v := value.(type) {
	case time.Time:
		return true
	case string:
		v = strings.TrimSpace(v)
		if v == "" || v == "Invalid Date" {
			return false
		}
		if v == "now()" {
			return true
		}
		for _, layout := range dateTimeLayouts() {
			if _, err := time.Parse(layout, v); err == nil {
				return true
			}
		}
	}
	return false
}

func dateTimeLayouts() []string {
	return []string{
		time.RFC3339Nano,
		"2006-01",
		"2006-01-02",
		"2006-01-02T15:04",
	}
}

func numericValue(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, !math.IsNaN(v) && !math.IsInf(v, 0)
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case jsonNumber:
		number, err := v.Float64()
		return number, err == nil
	}
	return 0, false
}

type jsonNumber interface{ Float64() (float64, error) }

func numericField(name string) bool {
	switch name {
	case "currentStockQty", "purchasePrice", "amount", "deductedAmount", "freightAmount", "advanceAmount", "otherCharges", "partialReceivedPayment", "cargoWeight":
		return true
	default:
		return false
	}
}

func enumValuesForCast(cast string) []string {
	switch cast {
	case "InventoryCategory":
		return []string{"Spare", "Fuel", "Goods", "Tools"}
	case "InventoryUnitType":
		return []string{"Litre", "KG", "Piece"}
	case "VehicleType":
		return []string{"Flatbed Trailer", "Low Bed Trailer", "Container Trailer", "Skeletal Trailer", "Curtain Side Trailer", "Refrigerated Trailer (Reefer)", "Fuel Tanker", "Water Tanker", "Dump Truck / Tipper", "Car Carrier", "17 ft Mazda", "22 ft Truck", "28 ft Truck", "Tractor Head"}
	case "VehicleStatus":
		return []string{"Available", "Avaliable", "On Trip", "Maintenance", "Out of Service"}
	case "LicenseType":
		return []string{"HTV", "PSV", "IDP", "LTV"}
	case "DriverStatus":
		return []string{"Available", "Avaliable", "On Leave", "Suspended", "On Trip"}
	case "VehicleMaintenanceCategory":
		return []string{"Engine", "Tires", "Brakes", "Electrical", "Suspension", "Transmission", "Oil & Fluids", "Body / General"}
	case "VehicleMaintenancePaymentStatus", "FuelPaymentStatus":
		return []string{"Paid", "Pending"}
	case "DamageType":
		return []string{"Broken", "Damaged Packaging", "Missing Items", "Scratched"}
	case "DeductionType":
		return []string{"Cancelled", "Late Delivery"}
	case "TripPaymentStatus":
		return []string{"Received", "Pending", "Partial Received", "Paid"}
	case "TourStatus":
		return []string{"Pre-Planned", "In Progress", "Completed", "Cancelled", "Late Delivery"}
	case "Role":
		return []string{"Super Admin", "Admin", "Manager", "User"}
	case "Status":
		return []string{"Active", "Inactive"}
	}
	return nil
}

func oneOf(value string, values []string) bool {
	for _, allowed := range values {
		if value == allowed {
			return true
		}
	}
	return false
}

func writeValidationErrors(w http.ResponseWriter, messages []string) bool {
	if len(messages) == 0 {
		return false
	}
	writeError(w, 400, messages)
	return true
}
