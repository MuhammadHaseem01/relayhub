package router

import "testing"

func TestNamedHTTPFormatsPartialJSONObject(t *testing.T) {
	got, ok := namedHTTP(`{"id":"3"}`).(map[string]any)
	if !ok {
		t.Fatalf("namedHTTP() = %#v, want map", got)
	}
	if got["id"] != "3" || got["name"] != "3" {
		t.Fatalf("namedHTTP() = %#v, want id/name 3", got)
	}
}

func TestNamedHTTPFormatsFullJSONObject(t *testing.T) {
	got, ok := namedHTTP(`{"id":"3","name":"Ali"}`).(map[string]any)
	if !ok {
		t.Fatalf("namedHTTP() = %#v, want map", got)
	}
	if got["id"] != "3" || got["name"] != "Ali" {
		t.Fatalf("namedHTTP() = %#v, want id 3 name Ali", got)
	}
}
