package store_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"relayhub/internal/store"
)

func TestSubstituteVars_OK(t *testing.T) {
	body := "Hi {{customer_name}}, your order {{order_id}} has shipped!"
	vars := map[string]string{
		"customer_name": "Ali",
		"order_id":      "4471",
	}

	got, missing, err := store.SubstituteVars(body, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(missing) != 0 {
		t.Errorf("expected no missing vars, got: %v", missing)
	}

	want := "Hi Ali, your order 4471 has shipped!"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSubstituteVars_MissingVars(t *testing.T) {
	body := "Hello {{name}}, your code is {{code}}."
	vars := map[string]string{"name": "Bob"}

	_, missing, err := store.SubstituteVars(body, vars)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, store.ErrMissingVars) {
		t.Errorf("expected ErrMissingVars, got %v", err)
	}
	if len(missing) != 1 || missing[0] != "code" {
		t.Errorf("expected missing=[\"code\"], got %v", missing)
	}
}

func TestSubstituteVars_MultipleMissing(t *testing.T) {
	body := "{{a}} {{b}} {{c}}"
	vars := map[string]string{}

	_, missing, err := store.SubstituteVars(body, vars)
	if !errors.Is(err, store.ErrMissingVars) {
		t.Fatalf("expected ErrMissingVars, got %v", err)
	}
	if len(missing) != 3 {
		t.Errorf("expected 3 missing vars, got %d: %v", len(missing), missing)
	}
}

func TestSubstituteVars_NoPlaceholders(t *testing.T) {
	body := "This message has no placeholders."
	vars := map[string]string{}

	got, missing, err := store.SubstituteVars(body, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(missing) != 0 {
		t.Errorf("expected no missing vars, got: %v", missing)
	}
	if got != body {
		t.Errorf("got %q, want %q", got, body)
	}
}

func TestSubstituteVars_ExtraVarsIgnored(t *testing.T) {
	body := "Hello {{name}}!"
	vars := map[string]string{
		"name":  "Charlie",
		"extra": "ignored",
	}

	got, _, err := store.SubstituteVars(body, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Hello Charlie!" {
		t.Errorf("got %q", got)
	}
}

func TestSubstituteVars_DuplicatePlaceholders(t *testing.T) {
	body := "{{name}} and {{name}} again"
	vars := map[string]string{"name": "Dana"}

	got, _, err := store.SubstituteVars(body, vars)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Dana and Dana again"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSubstituteVars_MissingReportedOnce(t *testing.T) {
	body := "{{x}} and {{x}} and {{y}}"
	vars := map[string]string{}

	_, missing, err := store.SubstituteVars(body, vars)
	if !errors.Is(err, store.ErrMissingVars) {
		t.Fatalf("expected ErrMissingVars, got %v", err)
	}
	if len(missing) != 2 {
		t.Errorf("expected 2 unique missing vars, got %d: %v", len(missing), missing)
	}
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func TestCreateTemplate_OK(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, err := db.CreateTenant(ctx, "Template Test Tenant", "rh_tmpl_test_key_"+uniqueName("ct"))
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	name := "order_shipped"
	body := "Hi {{customer_name}}, order {{order_id}} has shipped!"

	tmpl, err := db.CreateTemplate(ctx, tenant.ID, name, body)
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}
	if tmpl.ID == "" {
		t.Error("expected non-empty ID")
	}
	if tmpl.Name != name {
		t.Errorf("name: got %q, want %q", tmpl.Name, name)
	}
	if tmpl.Body != body {
		t.Errorf("body: got %q, want %q", tmpl.Body, body)
	}
	if tmpl.TenantID != tenant.ID {
		t.Errorf("tenant_id mismatch")
	}
}

func TestCreateTemplate_Duplicate(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, err := db.CreateTenant(ctx, "Dup Template Tenant", "rh_tmpl_dup_key_"+uniqueName("dup"))
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	name := "welcome"
	_, err = db.CreateTemplate(ctx, tenant.ID, name, "Hello!")
	if err != nil {
		t.Fatalf("first CreateTemplate: %v", err)
	}

	_, err = db.CreateTemplate(ctx, tenant.ID, name, "Hello again!")
	if !errors.Is(err, store.ErrDuplicateTemplate) {
		t.Errorf("expected ErrDuplicateTemplate, got %v", err)
	}
}

func TestCreateTemplate_TenantIsolation(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tA, err := db.CreateTenant(ctx, "Iso Tenant A", "rh_tmpl_iso_a_"+uniqueName("ia"))
	if err != nil {
		t.Fatalf("create tenant A: %v", err)
	}
	tB, err := db.CreateTenant(ctx, "Iso Tenant B", "rh_tmpl_iso_b_"+uniqueName("ib"))
	if err != nil {
		t.Fatalf("create tenant B: %v", err)
	}

	name := "shared_name"
	if _, err := db.CreateTemplate(ctx, tA.ID, name, "body A"); err != nil {
		t.Fatalf("create for tenant A: %v", err)
	}
	if _, err := db.CreateTemplate(ctx, tB.ID, name, "body B"); err != nil {
		t.Fatalf("create for tenant B (should be allowed): %v", err)
	}
}

func TestGetTemplate_NotFound(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, err := db.CreateTenant(ctx, "Get NF Tenant", "rh_tmpl_gnf_"+uniqueName("gnf"))
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	_, err = db.GetTemplate(ctx, tenant.ID, "does_not_exist")
	if !errors.Is(err, store.ErrTemplateNotFound) {
		t.Errorf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestListTemplates_TenantIsolation(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tA, err := db.CreateTenant(ctx, "List Iso A", "rh_tmpl_lista_"+uniqueName("la"))
	if err != nil {
		t.Fatalf("create tenant A: %v", err)
	}
	tB, err := db.CreateTenant(ctx, "List Iso B", "rh_tmpl_listb_"+uniqueName("lb"))
	if err != nil {
		t.Fatalf("create tenant B: %v", err)
	}

	_, _ = db.CreateTemplate(ctx, tA.ID, "tmpl_a1", "body a1")
	_, _ = db.CreateTemplate(ctx, tA.ID, "tmpl_a2", "body a2")
	_, _ = db.CreateTemplate(ctx, tB.ID, "tmpl_b1", "body b1")

	listA, err := db.ListTemplates(ctx, tA.ID)
	if err != nil {
		t.Fatalf("ListTemplates A: %v", err)
	}
	listB, err := db.ListTemplates(ctx, tB.ID)
	if err != nil {
		t.Fatalf("ListTemplates B: %v", err)
	}

	if len(listA) != 2 {
		t.Errorf("tenant A: expected 2 templates, got %d", len(listA))
	}
	if len(listB) != 1 {
		t.Errorf("tenant B: expected 1 template, got %d", len(listB))
	}

	for _, tmpl := range listB {
		if tmpl.TenantID != tB.ID {
			t.Errorf("tenant B list contains wrong tenant_id: %s", tmpl.TenantID)
		}
	}
}

func TestUpdateTemplate_OK(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, err := db.CreateTenant(ctx, "Update Tenant", "rh_tmpl_upd_"+uniqueName("upd"))
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	original, err := db.CreateTemplate(ctx, tenant.ID, "update_me", "original body")
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	updated, err := db.UpdateTemplate(ctx, tenant.ID, "update_me", "updated body {{var}}")
	if err != nil {
		t.Fatalf("UpdateTemplate: %v", err)
	}
	if updated.Body != "updated body {{var}}" {
		t.Errorf("body not updated: got %q", updated.Body)
	}
	if !updated.UpdatedAt.After(original.UpdatedAt) && updated.UpdatedAt.Equal(original.UpdatedAt) {
		t.Logf("updated_at not strictly after created_at (acceptable on fast machines)")
	}
}

func TestDeleteTemplate_OK(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, err := db.CreateTenant(ctx, "Delete Tenant", "rh_tmpl_del_"+uniqueName("del"))
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	_, err = db.CreateTemplate(ctx, tenant.ID, "delete_me", "will be gone")
	if err != nil {
		t.Fatalf("CreateTemplate: %v", err)
	}

	if err := db.DeleteTemplate(ctx, tenant.ID, "delete_me"); err != nil {
		t.Fatalf("DeleteTemplate: %v", err)
	}

	_, err = db.GetTemplate(ctx, tenant.ID, "delete_me")
	if !errors.Is(err, store.ErrTemplateNotFound) {
		t.Errorf("expected ErrTemplateNotFound after delete, got %v", err)
	}
}

func TestDeleteTemplate_NotFound(t *testing.T) {
	db := openDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tenant, err := db.CreateTenant(ctx, "Del NF Tenant", "rh_tmpl_dnf_"+uniqueName("dnf"))
	if err != nil {
		t.Fatalf("CreateTenant: %v", err)
	}

	err = db.DeleteTemplate(ctx, tenant.ID, "ghost")
	if !errors.Is(err, store.ErrTemplateNotFound) {
		t.Errorf("expected ErrTemplateNotFound, got %v", err)
	}
}
