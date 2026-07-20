package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"relayhub/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}
	db, err := store.New(url)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func TestTenantIsolation(t *testing.T) {
	db := openTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenantA, err := db.CreateTenant(ctx, "Tenant Alpha", "rh_test_alpha_isolation_key_001")
	if err != nil {
		t.Fatalf("create tenant A: %v", err)
	}
	tenantB, err := db.CreateTenant(ctx, "Tenant Beta", "rh_test_beta_isolation_key_002")
	if err != nil {
		t.Fatalf("create tenant B: %v", err)
	}

	if err := db.LogNotification(ctx, store.NotificationRecord{
		TenantID:  tenantA.ID,
		RequestID: "req-alpha-001",
		Recipient: "alpha@example.com",
		Channel:   "email",
		Message:   "Message for Alpha only",
		Status:    "delivered",
	}); err != nil {
		t.Fatalf("log notification for tenant A: %v", err)
	}

	if err := db.LogNotification(ctx, store.NotificationRecord{
		TenantID:  tenantB.ID,
		RequestID: "req-beta-001",
		Recipient: "beta@example.com",
		Channel:   "email",
		Message:   "Message for Beta only",
		Status:    "delivered",
	}); err != nil {
		t.Fatalf("log notification for tenant B: %v", err)
	}

	logsA, err := db.GetLogs(ctx, tenantA.ID, 50)
	if err != nil {
		t.Fatalf("GetLogs for tenant A: %v", err)
	}
	for _, l := range logsA {
		if l.TenantID != tenantA.ID {
			t.Errorf("tenant A log contains foreign record: tenant_id=%s (want %s)", l.TenantID, tenantA.ID)
		}
		if l.RequestID == "req-beta-001" {
			t.Error("tenant A must NOT see tenant B's notification (req-beta-001)")
		}
	}
	foundA := false
	for _, l := range logsA {
		if l.RequestID == "req-alpha-001" {
			foundA = true
		}
	}
	if !foundA {
		t.Error("tenant A should see its own notification (req-alpha-001)")
	}

	logsB, err := db.GetLogs(ctx, tenantB.ID, 50)
	if err != nil {
		t.Fatalf("GetLogs for tenant B: %v", err)
	}
	for _, l := range logsB {
		if l.TenantID != tenantB.ID {
			t.Errorf("tenant B log contains foreign record: tenant_id=%s (want %s)", l.TenantID, tenantB.ID)
		}
		if l.RequestID == "req-alpha-001" {
			t.Error("tenant B must NOT see tenant A's notification (req-alpha-001)")
		}
	}
	foundB := false
	for _, l := range logsB {
		if l.RequestID == "req-beta-001" {
			foundB = true
		}
	}
	if !foundB {
		t.Error("tenant B should see its own notification (req-beta-001)")
	}

	logsEmpty, err := db.GetLogs(ctx, "", 50)
	if err != nil {
		t.Fatalf("GetLogs with empty tenantID: %v", err)
	}
	if len(logsEmpty) != 0 {
		t.Errorf("empty tenantID should return no rows, got %d", len(logsEmpty))
	}
}
