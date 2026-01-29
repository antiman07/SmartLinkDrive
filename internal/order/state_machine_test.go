package order

import (
	"testing"
	"time"
)

func TestCanTransitionAndApply(t *testing.T) {
	if !CanTransition(StatusCreated, StatusAssigned) {
		t.Fatalf("expected created -> assigned allowed")
	}
	if CanTransition(StatusCompleted, StatusCreated) {
		t.Fatalf("expected completed -> created not allowed")
	}

	o := &Order{Status: StatusCreated}
	now := time.Now()
	if err := ApplyTransition(o, StatusAssigned, now); err != nil {
		t.Fatalf("ApplyTransition: %v", err)
	}
	if o.Status != StatusAssigned {
		t.Fatalf("expected status assigned, got %s", o.Status)
	}

	if err := ApplyTransition(o, StatusCompleted, now); err == nil {
		t.Fatalf("expected invalid shortcut transition to fail")
	}
}
