package store_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"relayhub/internal/store"
)

func openDB(t *testing.T) *store.Store {
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

func TestGetTenantUsage_ReflectsDBState(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, err := db.CreateTenant(ctx, "Usage Test Tenant", "rh_usage_test_key_integration_001")
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}
	usage, err := db.GetTenantUsage(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("GetTenantUsage (initial): %v", err)
	}
	if usage.Count != 0 {
		t.Errorf("expected 0 usage initially, got %d", usage.Count)
	}
	if usage.OldestAt != nil {
		t.Error("expected nil OldestAt when no notifications")
	}

	for i := 0; i < 3; i++ {
		if err := db.LogNotification(ctx, store.NotificationRecord{
			TenantID:  tenant.ID,
			RequestID: fmt.Sprintf("req-usage-%d", i),
			Recipient: "test@example.com",
			Channel:   "email",
			Message:   "usage test",
			Status:    "delivered",
		}); err != nil {
			t.Fatalf("LogNotification %d: %v", i, err)
		}
	}
	usage, err = db.GetTenantUsage(ctx, tenant.ID)
	if err != nil {
		t.Fatalf("GetTenantUsage (after logs): %v", err)
	}
	if usage.Count != 3 {
		t.Errorf("expected usage.Count=3, got %d", usage.Count)
	}
	if usage.OldestAt == nil {
		t.Error("expected non-nil OldestAt after logging notifications")
	}
}

func TestGetTenantUsage_CrossTenantIsolation(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tA, err := db.CreateTenant(ctx, "Usage Tenant A", "rh_usage_tenant_a_key_002")
	if err != nil {
		t.Fatalf("create tenant A: %v", err)
	}
	tB, err := db.CreateTenant(ctx, "Usage Tenant B", "rh_usage_tenant_b_key_003")
	if err != nil {
		t.Fatalf("create tenant B: %v", err)
	}
	for i := 0; i < 5; i++ {
		_ = db.LogNotification(ctx, store.NotificationRecord{
			TenantID: tA.ID, RequestID: fmt.Sprintf("rua-%d", i),
			Recipient: "a@example.com", Channel: "email",
			Message: "msg", Status: "delivered",
		})
	}

	usageA, _ := db.GetTenantUsage(ctx, tA.ID)
	usageB, _ := db.GetTenantUsage(ctx, tB.ID)

	if usageA.Count != 5 {
		t.Errorf("tenant A: expected 5, got %d", usageA.Count)
	}
	if usageB.Count != 0 {
		t.Errorf("tenant B should have 0 usage, got %d", usageB.Count)
	}
}
