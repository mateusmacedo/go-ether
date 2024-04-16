package identity_test

import (
	"testing"

	"github.com/mateusmacedo/goether/core/vo/identity"
)

func TestGenerateID(t *testing.T) {
	t.Run("GenerateID with INT type", func(t *testing.T) {
		sut := identity.NewIDGenerator()
		id, err := sut.GenerateID(identity.INT)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(id) == 0 {
			t.Errorf("expected non-empty ID, got empty")
		}
	})

	t.Run("GenerateID with STRING type", func(t *testing.T) {
		sut := identity.NewIDGenerator()
		id, err := sut.GenerateID(identity.STRING)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(id) != 10 {
			t.Errorf("expected ID length 10, got %d", len(id))
		}
	})

	t.Run("GenerateID with UUID type", func(t *testing.T) {
		sut := identity.NewIDGenerator()
		id, err := sut.GenerateID(identity.UUID)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(id) == 0 {
			t.Errorf("expected non-empty ID, got empty")
		}
	})

	t.Run("GenerateID with invalid type", func(t *testing.T) {
		sut := identity.NewIDGenerator()
		_, err := sut.GenerateID("invalid")
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
}