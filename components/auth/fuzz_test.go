package auth

import "testing"

// FuzzHashPassword fuzzes hashPassword with arbitrary passwords.
func FuzzHashPassword(f *testing.F) {
	seeds := []string{
		"password123",
		"",
		"a",
		"short",
		"very-long-password-with-many-characters-1234567890",
		"   spaces   ",
		"unicode-ñ-ü-ç",
		"special!@#$%^&*()_+-=[]{}|;':\",./<>?`~",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, password string) {
		hash, err := hashPassword(password)
		if err != nil {
			t.Errorf("hashPassword(%q) error: %v", password, err)
		}
		if hash == "" {
			t.Errorf("hashPassword(%q) returned empty hash", password)
		}
	})
}
