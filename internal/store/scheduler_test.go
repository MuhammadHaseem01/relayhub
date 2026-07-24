package store_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"relayhub/internal/store"
)

func createDueScheduled(t *testing.T, db *store.Store, tenantID, key string) store.NotificationRecord {
	t.Helper()
	pastTime := time.Now().Add(-1 * time.Minute)
	rec, err := db.CreateScheduled(context.Background(), store.NotificationRecord{
		TenantID:     tenantID,
		RequestID:    fmt.Sprintf("sched-%s-%d", key, time.Now().UnixNano()),
		Recipient:    "test@example.com",
		Channel:      "email",
		Message:      "scheduled message",
		ScheduledFor: &pastTime,
	})
	if err != nil {
		t.Fatalf("createDueScheduled: %v", err)
	}
	return rec
}

func createFutureScheduled(t *testing.T, db *store.Store, tenantID, key string) store.NotificationRecord {
	t.Helper()
	futureTime := time.Now().Add(24 * time.Hour)
	rec, err := db.CreateScheduled(context.Background(), store.NotificationRecord{
		TenantID:     tenantID,
		RequestID:    fmt.Sprintf("future-%s-%d", key, time.Now().UnixNano()),
		Recipient:    "test@example.com",
		Channel:      "email",
		Message:      "future message",
		ScheduledFor: &futureTime,
	})
	if err != nil {
		t.Fatalf("createFutureScheduled: %v", err)
	}
	return rec
}

func TestCreateScheduled_OK(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, err := db.CreateTenant(ctx, "Sched Tenant", "rh_sched_ok_"+uniqueName("sk"))
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	sendAt := time.Now().Add(5 * time.Minute)
	rec, err := db.CreateScheduled(ctx, store.NotificationRecord{
		TenantID:     tenant.ID,
		RequestID:    "sched-req-001",
		Recipient:    "test@example.com",
		Channel:      "email",
		Message:      "Hi {{name}}, shipped!",
		ScheduledFor: &sendAt,
	})
	if err != nil {
		t.Fatalf("CreateScheduled: %v", err)
	}
	if rec.Status != "scheduled" {
		t.Errorf("expected status=scheduled, got %q", rec.Status)
	}
	if rec.ScheduledFor == nil {
		t.Error("expected non-nil ScheduledFor")
	}
	if rec.RequestID != "sched-req-001" {
		t.Errorf("request_id mismatch: %s", rec.RequestID)
	}
}

func TestClaimDueNotifications_ClaimsOnlyDue(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, _ := db.CreateTenant(ctx, "Claim Tenant", "rh_claim_"+uniqueName("cl"))

	due := createDueScheduled(t, db, tenant.ID, "due")
	_ = createFutureScheduled(t, db, tenant.ID, "fut")

	claimed, err := db.ClaimDueNotifications(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimDueNotifications: %v", err)
	}

	foundDue := false
	for _, r := range claimed {
		if r.RequestID == due.RequestID {
			foundDue = true
			if r.Status != "processing" {
				t.Errorf("claimed row should have status=processing, got %q", r.Status)
			}
		}
	}
	if !foundDue {
		t.Error("due notification was not claimed")
	}

	reClaimed, err := db.ClaimDueNotifications(ctx, 10)
	if err != nil {
		t.Fatalf("re-claim: %v", err)
	}
	for _, r := range reClaimed {
		if r.RequestID == due.RequestID {
			t.Error("due notification was double-claimed")
		}
	}
}

func TestGetNotificationByRequestID_OK(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, _ := db.CreateTenant(ctx, "Get Notif Tenant", "rh_gn_"+uniqueName("gn"))
	rec := createDueScheduled(t, db, tenant.ID, "get")

	got, err := db.GetNotificationByRequestID(ctx, tenant.ID, rec.RequestID)
	if err != nil {
		t.Fatalf("GetNotificationByRequestID: %v", err)
	}
	if got.RequestID != rec.RequestID {
		t.Errorf("request_id mismatch")
	}
}

func TestGetNotificationByRequestID_TenantIsolation(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tA, _ := db.CreateTenant(ctx, "GN Iso A", "rh_gnia_"+uniqueName("gnia"))
	tB, _ := db.CreateTenant(ctx, "GN Iso B", "rh_gnib_"+uniqueName("gnib"))

	recA := createDueScheduled(t, db, tA.ID, "iso")

	_, err := db.GetNotificationByRequestID(ctx, tB.ID, recA.RequestID)
	if !errors.Is(err, store.ErrNotificationNotFound) {
		t.Errorf("expected ErrNotificationNotFound for cross-tenant lookup, got %v", err)
	}
}

func TestCancelScheduledNotification_OK(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, _ := db.CreateTenant(ctx, "Cancel Tenant", "rh_cancel_"+uniqueName("ca"))
	rec := createFutureScheduled(t, db, tenant.ID, "cancel")

	if err := db.CancelScheduledNotification(ctx, tenant.ID, rec.RequestID); err != nil {
		t.Fatalf("CancelScheduledNotification: %v", err)
	}

	got, err := db.GetNotificationByRequestID(ctx, tenant.ID, rec.RequestID)
	if err != nil {
		t.Fatalf("GetNotificationByRequestID after cancel: %v", err)
	}
	if got.Status != "cancelled" {
		t.Errorf("expected status=cancelled, got %q", got.Status)
	}
}

func TestCancelScheduledNotification_AlreadySent(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, _ := db.CreateTenant(ctx, "Cancel Sent Tenant", "rh_csent_"+uniqueName("cs"))
	due := createDueScheduled(t, db, tenant.ID, "alreadysent")

	_, _ = db.ClaimDueNotifications(ctx, 10)

	err := db.CancelScheduledNotification(ctx, tenant.ID, due.RequestID)
	if !errors.Is(err, store.ErrNotificationAlreadySent) {
		t.Errorf("expected ErrNotificationAlreadySent, got %v", err)
	}
}

func TestCancelScheduledNotification_NotFound(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, _ := db.CreateTenant(ctx, "Cancel NF Tenant", "rh_cnf_"+uniqueName("cnf"))

	err := db.CancelScheduledNotification(ctx, tenant.ID, "totally-fake-request-id")
	if !errors.Is(err, store.ErrNotificationNotFound) {
		t.Errorf("expected ErrNotificationNotFound, got %v", err)
	}
}

func TestUpdateNotificationStatus_OK(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, _ := db.CreateTenant(ctx, "UpdStatus Tenant", "rh_upds_"+uniqueName("us"))
	rec := createDueScheduled(t, db, tenant.ID, "upd")

	if err := db.UpdateNotificationStatus(ctx, rec.RequestID, "delivered", "", 1, false); err != nil {
		t.Fatalf("UpdateNotificationStatus: %v", err)
	}

	got, _ := db.GetNotificationByRequestID(ctx, tenant.ID, rec.RequestID)
	if got.Status != "delivered" {
		t.Errorf("expected status=delivered, got %q", got.Status)
	}
}
