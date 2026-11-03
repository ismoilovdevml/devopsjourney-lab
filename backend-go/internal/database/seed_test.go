package database

import (
	"strings"
	"testing"

	"github.com/devops-lab/backend/internal/auth"
)

func TestResolveAdminPassword(t *testing.T) {
	for _, weak := range []string{"", "admin123", "ADMIN123", "short", "admin", "aaaaaaaaaa"} {
		pw, generated, err := resolveAdminPassword(weak, "admin")
		if err != nil {
			t.Fatal(err)
		}
		if !generated || len(pw) != generatedAdminPasswordLen || pw == weak {
			t.Fatalf("%q: expected generated password, got %q generated=%v", weak, pw, generated)
		}
		if err := auth.ValidatePasswordStrength(pw, "admin"); err != nil {
			t.Fatalf("generated password fails policy: %v", err)
		}
		for _, r := range pw {
			if !strings.ContainsRune(passwordAlphabet, r) {
				t.Fatalf("unexpected char %q", r)
			}
		}
	}
	pw, generated, err := resolveAdminPassword("Str0ng-Admin-Pass!", "admin")
	if err != nil || generated || pw != "Str0ng-Admin-Pass!" {
		t.Fatalf("strong configured password not used: %q %v %v", pw, generated, err)
	}
	a, _, _ := resolveAdminPassword("", "admin")
	b, _, _ := resolveAdminPassword("", "admin")
	if a == b {
		t.Fatal("generated passwords must differ")
	}
}
