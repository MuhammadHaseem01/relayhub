package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"
)

type contractCase struct {
	Name         string
	Method       string
	Path         string
	Body         any
	AuthRequired bool
}

type endpointResult struct {
	Status int
	JSON   any
	Raw    string
}

type backend struct {
	Name    string
	BaseURL string
	Token   string
}

func main() {
	nest := backend{Name: "nestjs", BaseURL: env("NEST_BASE_URL", "http://localhost:5000")}
	goapi := backend{Name: "go", BaseURL: env("GO_BASE_URL", "http://localhost:5001")}

	sharedToken := os.Getenv("CONTRACT_AUTH_TOKEN")
	if sharedToken != "" {
		nest.Token = sharedToken
		goapi.Token = sharedToken
	} else if os.Getenv("CONTRACT_LOGIN_EMAIL") != "" || os.Getenv("CONTRACT_LOGIN_PASSWORD") != "" {
		var err error
		nest.Token, err = login(nest)
		if err != nil {
			fatalf("login against %s failed: %v", nest.Name, err)
		}
		goapi.Token, err = login(goapi)
		if err != nil {
			fatalf("login against %s failed: %v", goapi.Name, err)
		}
	}

	cases := defaultCases()
	if nest.Token == "" || goapi.Token == "" {
		cases = publicCases(cases)
		fmt.Println("No CONTRACT_AUTH_TOKEN or CONTRACT_LOGIN_EMAIL/PASSWORD provided; running public cases only.")
	}

	failures := 0
	for _, tc := range cases {
		nestResult, err := request(nest, tc)
		if err != nil {
			failures++
			fmt.Printf("FAIL %-36s %s request error: %v\n", tc.Name, nest.Name, err)
			continue
		}
		goResult, err := request(goapi, tc)
		if err != nil {
			failures++
			fmt.Printf("FAIL %-36s %s request error: %v\n", tc.Name, goapi.Name, err)
			continue
		}
		if !sameResult(nestResult, goResult) {
			failures++
			fmt.Printf("FAIL %-36s status %d vs %d\n", tc.Name, nestResult.Status, goResult.Status)
			fmt.Printf("  nest: %s\n", compact(nestResult))
			fmt.Printf("  go:   %s\n", compact(goResult))
			continue
		}
		fmt.Printf("PASS %-36s %d\n", tc.Name, nestResult.Status)
	}

	if failures > 0 {
		fatalf("%d contract case(s) failed", failures)
	}
	fmt.Printf("All %d contract case(s) passed.\n", len(cases))
}

func defaultCases() []contractCase {
	return []contractCase{
		{Name: "root", Method: http.MethodGet, Path: "/api"},
		{Name: "cors preflight", Method: http.MethodOptions, Path: "/api/client"},
		{Name: "clients list unauthorized", Method: http.MethodGet, Path: "/api/client"},
		{Name: "login empty body validation", Method: http.MethodPost, Path: "/api/user/login", Body: map[string]any{}},
		{Name: "users list", Method: http.MethodGet, Path: "/api/user", AuthRequired: true},
		{Name: "users invalid pagination", Method: http.MethodGet, Path: "/api/user?page=0", AuthRequired: true},
		{Name: "clients list", Method: http.MethodGet, Path: "/api/client", AuthRequired: true},
		{Name: "clients list trailing slash", Method: http.MethodGet, Path: "/api/client/", AuthRequired: true},
		{Name: "client create empty validation", Method: http.MethodPost, Path: "/api/client/add", Body: map[string]any{}, AuthRequired: true},
		{Name: "client unknown field validation", Method: http.MethodPost, Path: "/api/client/add", Body: map[string]any{"unknown": "field"}, AuthRequired: true},
		{Name: "arrange vehicles list", Method: http.MethodGet, Path: "/api/arrange-vehicle", AuthRequired: true},
		{Name: "inventory list", Method: http.MethodGet, Path: "/api/inventory", AuthRequired: true},
		{Name: "inventory invalid enum validation", Method: http.MethodPost, Path: "/api/inventory/add", Body: map[string]any{"itemName": "Oil", "category": "Invalid", "unitType": "Litre", "currentStockQty": 1, "purchasePrice": 1}, AuthRequired: true},
		{Name: "vehicles list", Method: http.MethodGet, Path: "/api/vehicle", AuthRequired: true},
		{Name: "drivers list", Method: http.MethodGet, Path: "/api/driver", AuthRequired: true},
		{Name: "vehicle maintenance list", Method: http.MethodGet, Path: "/api/vehicle-maintenance", AuthRequired: true},
		{Name: "vehicle maintenance update missing id", Method: http.MethodPatch, Path: "/api/vehicle-maintenance/update", Body: map[string]any{}, AuthRequired: true},
		{Name: "tour damages list", Method: http.MethodGet, Path: "/api/tour-damage", AuthRequired: true},
		{Name: "tour deductions list", Method: http.MethodGet, Path: "/api/tour-deduction", AuthRequired: true},
		{Name: "tours list", Method: http.MethodGet, Path: "/api/tour", AuthRequired: true},
		{Name: "tour petrol pump invalid update", Method: http.MethodPatch, Path: "/api/tour/not-a-pump", Body: map[string]any{}, AuthRequired: true},
		{Name: "ledgers list", Method: http.MethodGet, Path: "/api/ledger", AuthRequired: true},
		{Name: "ledger undefined alias", Method: http.MethodGet, Path: "/api/ledger/undefined", AuthRequired: true},
	}
}

func publicCases(cases []contractCase) []contractCase {
	out := []contractCase{}
	for _, tc := range cases {
		if !tc.AuthRequired {
			out = append(out, tc)
		}
	}
	return out
}

func login(b backend) (string, error) {
	body := map[string]any{
		"email":    os.Getenv("CONTRACT_LOGIN_EMAIL"),
		"password": os.Getenv("CONTRACT_LOGIN_PASSWORD"),
	}
	result, err := request(b, contractCase{Name: "login", Method: http.MethodPost, Path: "/api/user/login", Body: body})
	if err != nil {
		return "", err
	}
	if result.Status < 200 || result.Status >= 300 {
		return "", fmt.Errorf("status %d: %s", result.Status, result.Raw)
	}
	object, ok := result.JSON.(map[string]any)
	if !ok {
		return "", errors.New("login response is not a JSON object")
	}
	token, _ := object["Token"].(string)
	if token == "" {
		return "", errors.New("login response missing Token")
	}
	return token, nil
}

func request(b backend, tc contractCase) (endpointResult, error) {
	var body io.Reader
	if tc.Body != nil {
		raw, err := json.Marshal(tc.Body)
		if err != nil {
			return endpointResult{}, err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(tc.Method, strings.TrimRight(b.BaseURL, "/")+tc.Path, body)
	if err != nil {
		return endpointResult{}, err
	}
	if tc.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if tc.AuthRequired && b.Token != "" {
		req.Header.Set("Authorization", "Bearer "+b.Token)
	}
	client := http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return endpointResult{}, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return endpointResult{}, err
	}
	result := endpointResult{Status: resp.StatusCode, Raw: string(raw)}
	if len(raw) > 0 {
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err == nil {
			result.JSON = decoded
		}
	}
	return result, nil
}

func sameResult(a endpointResult, b endpointResult) bool {
	if a.Status != b.Status {
		return false
	}
	if a.JSON != nil || b.JSON != nil {
		return reflect.DeepEqual(normalize(a.JSON), normalize(b.JSON))
	}
	return strings.TrimSpace(a.Raw) == strings.TrimSpace(b.Raw)
}

func normalize(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, item := range v {
			out[key] = normalize(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = normalize(item)
		}
		return out
	case float64, string, bool, nil:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func compact(result endpointResult) string {
	if result.JSON != nil {
		raw, err := json.Marshal(result.JSON)
		if err == nil {
			return string(raw)
		}
	}
	return strings.TrimSpace(result.Raw)
}

func env(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
