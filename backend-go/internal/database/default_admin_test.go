package database

import (
	"testing"

	"github.com/devops-lab/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// cheapHash keeps the test fast; CompareHashAndPassword reads the cost from the hash.
func cheapHash(t *testing.T, pw string) string {
	t.Helper()
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestMatchesDefaultPassword(t *testing.T) {
	for _, pw := range KnownDefaultPasswords {
		if got, ok := matchesDefaultPassword(cheapHash(t, pw)); !ok || got != pw {
			t.Errorf("%q not detected (got %q, %v)", pw, got, ok)
		}
	}
	for _, pw := range []string{"Str0ng-Admin-Pass!", "Admin123", "admin1234"} {
		if got, ok := matchesDefaultPassword(cheapHash(t, pw)); ok {
			t.Errorf("%q falsely matched %q", pw, got)
		}
	}
	if _, ok := matchesDefaultPassword(""); ok {
		t.Error("empty hash must not match")
	}
}

func TestAdminsWithDefaultPassword(t *testing.T) {
	admins := []models.User{
		{Username: "root", Password: cheapHash(t, "admin123")},
		{Username: "ops", Password: cheapHash(t, "Str0ng-Admin-Pass!")},
		{Username: "legacy", Password: cheapHash(t, "changeme")},
	}
	got := adminsWithDefaultPassword(admins)
	if len(got) != 2 || got[0] != "root" || got[1] != "legacy" {
		t.Fatalf("got %v, want [root legacy]", got)
	}
	if got := adminsWithDefaultPassword(nil); len(got) != 0 {
		t.Fatalf("no admins: %v", got)
	}
}
