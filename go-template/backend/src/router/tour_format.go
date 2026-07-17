package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (s *Server) formatTourList(r *http.Request, user currentUser, items []map[string]any) ([]map[string]any, error) {
	organizationID, organizationName, _ := s.store.UserOrganization(r.Context(), user.ID)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, formatTourItem(item, organizationID, organizationName))
	}
	return out, nil
}

func formatTourItem(tour map[string]any, organizationID string, _ string) map[string]any {
	expenses := jsonArray(tour["expenses"])
	fuelDetails := jsonArray(tour["fuelDetails"])
	totalExpenses := totalAmountHTTP(expenses)
	totalFuelExpenses := totalAmountHTTP(fuelDetails)
	freight := numberHTTP(tour["freightAmount"])
	startDate := timeHTTP(tour["startDate"])
	expectedEndDate := timeHTTP(tour["expectedEndDate"])
	createdAt := timeHTTP(tour["createdAt"])
	actualEndDate := ""
	if v := timeHTTP(tour["actualEndDate"]); !v.IsZero() {
		actualEndDate = v.Format("2006-01-02")
	}
	actualEndTime := ""
	if s := stringHTTP(tour["actualEndTime"]); s != "" {
		actualEndTime = formatClock(s)
	}
	formattedFuel := make([]map[string]any, 0, len(fuelDetails))
	for _, fuel := range fuelDetails {
		formattedFuel = append(formattedFuel, formatFuelDetailHTTP(fuel))
	}
	createdBy := ""
	if id := idFromItem(tour); id == 0 {
		_ = id
	}
	if v := tour["createdById"]; v != nil {
		createdBy = fmt.Sprint(v)
	}
	return map[string]any{
		"id": tour["id"], "tourName": tour["tourName"], "driver": namedHTTP(tour["driver"]), "vehicle": namedHTTP(tour["vehicle"]),
		"client": clientNamedHTTP(tour["client"]), "startLocation": jsonObject(tour["startLocation"]), "endLocation": jsonObject(tour["endLocation"]),
		"startDate": startDate.Format("2006-01-02"), "time": formatClock(stringHTTP(tour["time"])), "expectedEndDate": expectedDateHTTP(startDate, expectedEndDate),
		"actualEndDate": actualEndDate, "actualEndTime": actualEndTime, "freightAmount": freight, "advanceAmount": numberHTTP(tour["advanceAmount"]),
		"otherCharges": numberHTTP(tour["otherCharges"]), "paymentStatus": stringHTTP(tour["paymentStatus"]), "partialReceivedPayment": numberHTTP(tour["partialReceivedPayment"]),
		"expenses": expenses, "fuelDetails": formattedFuel, "loadType": stringHTTP(tour["loadType"]), "cargoWeight": numberHTTP(tour["cargoWeight"]),
		"vehicleType": stringHTTP(tour["vehicleType"]), "status": stringHTTP(tour["status"]), "notes": stringHTTP(tour["notes"]), "createdBy": createdBy,
		"organizationId": organizationID, "organizationName": "", "tourNumber": fmt.Sprintf("TOUR-%d-%v", createdAt.UnixMilli(), tour["id"]),
		"createdAt": tour["createdAt"], "updatedAt": tour["updatedAt"], "__v": 0, "totalFuelExpenses": totalFuelExpenses,
		"totalExpenses": totalExpenses, "Profit": freight - totalFuelExpenses - totalExpenses,
	}
}

func jsonObject(value any) map[string]any {
	obj := map[string]any{}
	_ = json.Unmarshal([]byte(stringHTTP(value)), &obj)
	return obj
}
func jsonArray(value any) []map[string]any {
	arr := []map[string]any{}
	_ = json.Unmarshal([]byte(stringHTTP(value)), &arr)
	return arr
}

func namedHTTP(value any) any {
	obj := jsonObject(value)
	id := stringHTTP(obj["id"])
	name := stringHTTP(obj["name"])
	if id != "" || name != "" {
		if name == "" {
			name = id
		}
		if id == "" {
			id = name
		}
		return map[string]any{"id": id, "name": name}
	}
	s := stringHTTP(value)
	if s == "" {
		return nil
	}
	return map[string]any{"id": s, "name": s}
}

func clientNamedHTTP(value any) any {
	if named := namedHTTP(value); named != nil {
		return named
	}
	s := stringHTTP(value)
	return map[string]any{"id": s, "name": s}
}

func expectedDateHTTP(start time.Time, expected time.Time) any {
	if expected.IsZero() || expected.Equal(start) {
		return nil
	}
	return expected.Format("2006-01-02")
}

func formatFuelDetailHTTP(fuel map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range fuel {
		out[key] = value
	}
	pumpName := stringHTTP(fuel["petrolPumpName"])
	id := stringHTTP(fuel["_id"])
	if id == "" {
		id = stringHTTP(fuel["id"])
	}
	if id == "" {
		id = pumpName
	}
	out["petrolPump"] = map[string]any{"id": id, "name": pumpName}
	out["_id"] = id
	return out
}

func totalAmountHTTP(items []map[string]any) float64 {
	total := 0.0
	for _, item := range items {
		total += numberHTTP(item["amount"])
	}
	return total
}

func numberHTTP(value any) float64 {
	switch v := value.(type) {
	case nil:
		return 0
	case float64:
		return v
	case int64:
		return float64(v)
	case int:
		return float64(v)
	case string:
		var n float64
		_, _ = fmt.Sscan(v, &n)
		return n
	}
	return 0
}
func stringHTTP(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}
func timeHTTP(value any) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case string:
		t, _ := time.Parse(time.RFC3339, v)
		return t
	}
	return time.Time{}
}

func formatClock(value string) string {
	if len(value) >= 5 {
		value = value[:5]
	}
	var hour, minute int
	if _, err := fmt.Sscanf(value, "%d:%d", &hour, &minute); err != nil {
		return value
	}
	suffix := "AM"
	if hour >= 12 {
		suffix = "PM"
	}
	display := hour % 12
	if display == 0 {
		display = 12
	}
	return fmt.Sprintf("%d:%02d %s", display, minute, suffix)
}
