package base

import (
	"testing"

	"github.com/kaiorocha/middleware-boletos/backend/internal/providers/types"
)

func TestStateMachineAllowsExpectedTransitions(t *testing.T) {
	if !CanTransition(types.StatusCreated, types.StatusProcessing) {
		t.Fatal("expected CREATED -> PROCESSING to be valid")
	}
	if !CanTransition(types.StatusProcessing, types.StatusIssued) {
		t.Fatal("expected PROCESSING -> ISSUED to be valid")
	}
	if !CanTransition(types.StatusIssued, types.StatusPaid) {
		t.Fatal("expected ISSUED -> PAID to be valid")
	}
}

func TestStateMachineRejectsInvalidTransitions(t *testing.T) {
	if CanTransition(types.StatusCreated, types.StatusPaid) {
		t.Fatal("expected CREATED -> PAID to be invalid")
	}
	if CanTransition(types.StatusPaid, types.StatusIssued) {
		t.Fatal("expected PAID -> ISSUED to be invalid")
	}
}
