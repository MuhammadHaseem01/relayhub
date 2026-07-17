package router

import "testing"

func TestRelationIDNameMapWithoutName(t *testing.T) {
	id, name := relationIDName(map[string]any{"id": "3"})
	if id != "3" || name != "" {
		t.Fatalf("relationIDName() = %q, %q; want 3, empty", id, name)
	}
}

func TestRelationIDNameJSONWithName(t *testing.T) {
	id, name := relationIDName(`{"id":"2","name":"Ali"}`)
	if id != "2" || name != "Ali" {
		t.Fatalf("relationIDName() = %q, %q; want 2, Ali", id, name)
	}
}

func TestStripTourUpdateReadOnlyFields(t *testing.T) {
	body := map[string]any{"id": 2, "createdAt": "x", "__v": 0, "tourName": "Trip"}
	stripTourUpdateReadOnlyFields(body)
	if body["id"] != nil || body["createdAt"] != nil || body["__v"] != nil {
		t.Fatalf("read-only fields were not stripped: %#v", body)
	}
	if body["tourName"] != "Trip" {
		t.Fatalf("editable field was removed: %#v", body)
	}
}

func TestKeepOnlyTourCompletionStatus(t *testing.T) {
	body := map[string]any{"status": "Completed", "tourName": "", "driver": map[string]any{"id": "2"}}
	keepOnlyTourCompletionStatus(body)
	if len(body) != 1 || body["status"] != "Completed" {
		t.Fatalf("completion update should keep only status, got %#v", body)
	}
}
