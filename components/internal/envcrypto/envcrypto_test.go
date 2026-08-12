package envcrypto_test

import (
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/internal/envcrypto"
)

// validKey is a 32-byte AES-256 key (64 hex chars).
const validKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func mustKey(t *testing.T, hex string) []byte {
	t.Helper()
	k, err := envcrypto.ParseKey(hex)
	if err != nil {
		t.Fatalf("ParseKey(%q): %v", hex, err)
	}
	return k
}

func TestParseKey(t *testing.T) {
	tests := []struct {
		name    string
		hex     string
		wantNil bool
		wantErr bool
	}{
		{name: "empty disables encryption", hex: "", wantNil: true},
		{name: "valid 32-byte key", hex: validKey},
		{name: "non-hex", hex: "zzzz", wantErr: true},
		{name: "wrong length (16 bytes)", hex: strings.Repeat("ab", 16), wantErr: true},
		{name: "wrong length (33 bytes)", hex: strings.Repeat("ab", 33), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := envcrypto.ParseKey(tt.hex)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got key=%v", key)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNil && key != nil {
				t.Fatalf("expected nil key, got %v", key)
			}
			if !tt.wantNil && len(key) != 32 {
				t.Fatalf("expected 32-byte key, got %d", len(key))
			}
		})
	}
}

func TestParseRootKey(t *testing.T) {
	raw := strings.Repeat("A", 43) + "="
	key, err := envcrypto.ParseRootKey(raw)
	if err != nil || len(key) != 32 {
		t.Fatalf("valid base64 root key: len=%d err=%v", len(key), err)
	}
	for _, input := range []string{"", "not-base64", "AA=="} {
		if _, err := envcrypto.ParseRootKey(input); err == nil {
			t.Fatalf("ParseRootKey(%q) accepted invalid configuration", input)
		}
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key := mustKey(t, validKey)
	plaintexts := []string{"", "x", "secret-token", "postgres://user:pass@host/db", "ünïcödé-π"}
	for _, pt := range plaintexts {
		t.Run(pt, func(t *testing.T) {
			enc, err := envcrypto.Encrypt(key, pt)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}
			if pt != "" && enc == pt {
				t.Fatal("ciphertext equals plaintext — not encrypted")
			}
			dec, err := envcrypto.Decrypt(key, enc)
			if err != nil {
				t.Fatalf("Decrypt: %v", err)
			}
			if dec != pt {
				t.Fatalf("round-trip mismatch: got %q want %q", dec, pt)
			}
		})
	}
}

func TestEncryptNonceIsRandom(t *testing.T) {
	key := mustKey(t, validKey)
	a, _ := envcrypto.Encrypt(key, "same-value")
	b, _ := envcrypto.Encrypt(key, "same-value")
	if a == b {
		t.Fatal("two encryptions of the same plaintext produced identical ciphertext — nonce not random")
	}
}

func TestNoOpModeWhenKeyNil(t *testing.T) {
	enc, err := envcrypto.Encrypt(nil, "plaintext")
	if err != nil || enc != "plaintext" {
		t.Fatalf("Encrypt(nil) = %q, %v; want pass-through", enc, err)
	}
	dec, err := envcrypto.Decrypt(nil, "plaintext")
	if err != nil || dec != "plaintext" {
		t.Fatalf("Decrypt(nil) = %q, %v; want pass-through", dec, err)
	}
}

func TestDecryptFailures(t *testing.T) {
	key := mustKey(t, validKey)
	enc, _ := envcrypto.Encrypt(key, "secret")

	tests := []struct {
		name   string
		stored string
	}{
		{name: "invalid base64", stored: "!!!not-base64!!!"},
		{name: "too short for nonce", stored: "AAAA"},
		{name: "tampered ciphertext", stored: enc[:len(enc)-2] + "AA"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := envcrypto.Decrypt(key, tt.stored); err == nil {
				t.Fatal("expected decrypt error, got nil")
			}
		})
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	enc, _ := envcrypto.Encrypt(mustKey(t, validKey), "secret")
	otherKey := mustKey(t, strings.Repeat("ff", 32))
	if _, err := envcrypto.Decrypt(otherKey, enc); err == nil {
		t.Fatal("expected GCM auth failure decrypting with the wrong key, got nil")
	}
}

func TestMaskValue(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "", want: "••••"},
		{name: "exactly 4 chars", value: "abcd", want: "••••"},
		{name: "five chars reveals last four", value: "abcde", want: "••••bcde"},
		{name: "long value", value: "secret-token", want: "••••oken"},
		// Last 4 runes are multibyte; byte-slicing would corrupt UTF-8 here.
		{name: "unicode tail", value: "key-café", want: "••••café"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := envcrypto.MaskValue(tt.value); got != tt.want {
				t.Fatalf("MaskValue(%q) = %q, want %q", tt.value, got, tt.want)
			}
		})
	}
}
