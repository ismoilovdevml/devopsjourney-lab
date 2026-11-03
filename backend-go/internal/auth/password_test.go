package auth

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashAndCheckPassword(t *testing.T) {
	h, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if cost, _ := bcrypt.Cost([]byte(h)); cost != 12 {
		t.Fatalf("bcrypt cost = %d, want 12", cost)
	}
	if !CheckPassword("correct horse battery", h) {
		t.Fatal("correct password rejected")
	}
	if CheckPassword("wrong password", h) {
		t.Fatal("wrong password accepted")
	}
	if CheckPassword("anything", "") {
		t.Fatal("empty hash accepted")
	}
}

func TestHashPasswordRejectsOver72Bytes(t *testing.T) {
	if _, err := HashPassword(strings.Repeat("a", 73)); !errors.Is(err, ErrPasswordTooLong) {
		t.Fatalf("want ErrPasswordTooLong, got %v", err)
	}
	if CheckPassword(strings.Repeat("a", 73), "$2a$12$abcdefghijklmnopqrstuuJ0Zl0fK7o1q2ZP4bS2ZQ6P9m8H4i7W.") {
		t.Fatal("over-long password must never match")
	}
}

func TestValidatePasswordStrength(t *testing.T) {
	cases := []struct {
		pw, user string
		want     error
	}{
		{"Tr0ub4dor&3x", "alice", nil},
		{"short1!", "alice", ErrPasswordTooShort},
		{strings.Repeat("ab", 37), "alice", ErrPasswordTooLong}, // 74 bytes
		{strings.Repeat("a", 72), "alice", ErrPasswordWeak},
		{"aaaaaaaa", "alice", ErrPasswordWeak},
		{"alice_wonder", "alice_wonder", ErrPasswordIsUsername},
		{"ALICE_WONDER", "alice_wonder", ErrPasswordIsUsername},
		{"admin123", "root", ErrPasswordWeak},
		{"Password", "root", ErrPasswordWeak},
		{"12345678", "root", ErrPasswordWeak},
		{"пароль-надёжный", "bob", nil}, // multi-byte chars count as characters
	}
	for _, tc := range cases {
		got := ValidatePasswordStrength(tc.pw, tc.user)
		if !errors.Is(got, tc.want) { // errors.Is(nil, nil) is true
			t.Errorf("ValidatePasswordStrength(%q,%q) = %v, want %v", tc.pw, tc.user, got, tc.want)
		}
	}
}

func TestValidateUsername(t *testing.T) {
	good := []string{"abc", "alice", "bob_99", "a-b-c", strings.Repeat("x", 32)}
	bad := []string{"", "ab", "Alice", "alice!", "al ice", "алиса", strings.Repeat("x", 33), "../etc"}
	for _, u := range good {
		if err := ValidateUsername(u); err != nil {
			t.Errorf("ValidateUsername(%q) unexpected error %v", u, err)
		}
	}
	for _, u := range bad {
		if err := ValidateUsername(u); err == nil {
			t.Errorf("ValidateUsername(%q) expected error", u)
		}
	}
}
