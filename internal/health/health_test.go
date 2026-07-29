package health_test

import (
	"sync"
	"testing"
	"time"

	"relayhub/internal/health"
)

func TestHealthRegistry_InitialState(t *testing.T) {
	reg := health.NewRegistry([]string{"discord", "email", "smtp"})

	if !reg.IsHealthy("discord") || !reg.IsHealthy("email") || !reg.IsHealthy("smtp") {
		t.Error("expected all providers to be healthy initially")
	}

	snap := reg.Snapshot()
	if snap["discord"] != "healthy" || snap["email"] != "healthy" || snap["smtp"] != "healthy" {
		t.Errorf("unexpected snapshot: %v", snap)
	}
}

func TestHealthRegistry_CircuitBreaker_Flow(t *testing.T) {
	openDuration := 50 * time.Millisecond
	reg := health.NewRegistryWithConfig([]string{"discord"}, 5, openDuration)

	for i := 0; i < 4; i++ {
		reg.RecordFailure("discord")
		if !reg.IsHealthy("discord") {
			t.Fatalf("expected healthy after %d failures", i+1)
		}
	}

	reg.RecordFailure("discord")
	if reg.IsHealthy("discord") {
		t.Fatal("expected unhealthy after 5 failures")
	}

	snap := reg.Snapshot()
	if snap["discord"] != "unhealthy" {
		t.Fatalf("expected snapshot to show unhealthy, got %q", snap["discord"])
	}

	time.Sleep(openDuration + 10*time.Millisecond)

	if !reg.IsHealthy("discord") {
		t.Fatal("expected IsHealthy=true for half-open trial request")
	}

	reg.RecordSuccess("discord")
	if !reg.IsHealthy("discord") {
		t.Fatal("expected healthy after trial success")
	}

	snap2 := reg.Snapshot()
	if snap2["discord"] != "healthy" {
		t.Fatalf("expected snapshot healthy, got %q", snap2["discord"])
	}
}

func TestHealthRegistry_HalfOpenFailure_ResetsTimer(t *testing.T) {
	openDuration := 50 * time.Millisecond
	reg := health.NewRegistryWithConfig([]string{"email"}, 3, openDuration)

	for i := 0; i < 3; i++ {
		reg.RecordFailure("email")
	}

	if reg.IsHealthy("email") {
		t.Fatal("expected unhealthy after 3 failures")
	}

	time.Sleep(openDuration + 10*time.Millisecond)

	if !reg.IsHealthy("email") {
		t.Fatal("expected half-open trial allowed")
	}
	reg.RecordFailure("email")
	if reg.IsHealthy("email") {
		t.Fatal("expected unhealthy after failed trial request")
	}

	snap := reg.Snapshot()
	if snap["email"] != "unhealthy" {
		t.Fatalf("expected unhealthy snapshot, got %q", snap["email"])
	}
}

func TestHealthRegistry_Concurrency(t *testing.T) {
	reg := health.NewRegistry([]string{"discord", "email"})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			reg.IsHealthy("discord")
		}()
		go func() {
			defer wg.Done()
			reg.RecordFailure("discord")
		}()
		go func() {
			defer wg.Done()
			reg.RecordSuccess("discord")
		}()
	}
	wg.Wait()
}
