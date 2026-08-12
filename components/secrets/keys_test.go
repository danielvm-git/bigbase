package secrets_test

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/danielvm/bigbase/components/secrets"
)

func mustHierarchy(t *testing.T, rootKey []byte) *secrets.KeyHierarchy {
	t.Helper()
	kh, err := secrets.NewKeyHierarchy(rootKey)
	if err != nil {
		t.Fatalf("NewKeyHierarchy: %v", err)
	}
	return kh
}

func testScope() secrets.Scope {
	return secrets.Scope{
		ProjectID:     "proj-00000000000000000000000000000001",
		EnvironmentID: "env-00000000000000000000000000000002",
		FolderID:      "fld-00000000000000000000000000000003",
		SecretID:      "sec-00000000000000000000000000000004",
		Version:       1,
	}
}

func TestKeyHierarchyRejectsInvalidRootKey(t *testing.T) {
	for _, tt := range []struct {
		name string
		key  []byte
	}{
		{name: "nil", key: nil},
		{name: "empty", key: []byte{}},
		{name: "16 bytes", key: bytes.Repeat([]byte{1}, 16)},
		{name: "33 bytes", key: bytes.Repeat([]byte{1}, 33)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := secrets.NewKeyHierarchy(tt.key); err == nil {
				t.Fatalf("NewKeyHierarchy accepted %s root key", tt.name)
			}
		})
	}
}

func TestKeyHierarchyProjectDataKeyRoundTrip(t *testing.T) {
	kh := mustHierarchy(t, testRootKey(t))
	keyID, algorithm, wrapped, err := kh.GenerateProjectDataKey("project-1")
	if err != nil {
		t.Fatalf("GenerateProjectDataKey: %v", err)
	}
	if keyID == "" || len(keyID) != 32 {
		t.Fatalf("unexpected key id %q", keyID)
	}
	if algorithm != "aes-256-gcm" {
		t.Fatalf("unexpected algorithm %q", algorithm)
	}
	if wrapped == "" || wrapped == keyID {
		t.Fatalf("wrapped key looks like plaintext: %q", wrapped)
	}
	dataKey, err := kh.UnwrapProjectDataKey("project-1", keyID, algorithm, wrapped)
	if err != nil {
		t.Fatalf("UnwrapProjectDataKey: %v", err)
	}
	if len(dataKey) != 32 {
		t.Fatalf("data key length %d", len(dataKey))
	}
	// Wrapping is a real AES-GCM envelope: the wrapped blob must not contain
	// the raw key bytes.
	if bytes.Contains(mustDecodeB64(t, wrapped), dataKey) {
		t.Fatal("wrapped blob contains raw data key bytes")
	}
}

func TestKeyHierarchyWrongRootKeyFails(t *testing.T) {
	khA := mustHierarchy(t, bytes.Repeat([]byte{0x11}, 32))
	khB := mustHierarchy(t, bytes.Repeat([]byte{0x22}, 32))
	keyID, algorithm, wrapped, err := khA.GenerateProjectDataKey("project-1")
	if err != nil {
		t.Fatalf("GenerateProjectDataKey: %v", err)
	}
	if _, err := khB.UnwrapProjectDataKey("project-1", keyID, algorithm, wrapped); err == nil {
		t.Fatal("wrong root key unwrapped the data key")
	}
}

func TestKeyHierarchyRejectsUnsupportedAlgorithm(t *testing.T) {
	kh := mustHierarchy(t, testRootKey(t))
	keyID, _, wrapped, err := kh.GenerateProjectDataKey("project-1")
	if err != nil {
		t.Fatalf("GenerateProjectDataKey: %v", err)
	}
	if _, err := kh.UnwrapProjectDataKey("project-1", keyID, "aes-128-gcm", wrapped); err == nil {
		t.Fatal("unsupported algorithm accepted")
	}
	if _, err := kh.UnwrapProjectDataKey("project-1", keyID, "", wrapped); err == nil {
		t.Fatal("empty algorithm accepted")
	}
}

func TestKeyHierarchyWrapIsProjectBound(t *testing.T) {
	kh := mustHierarchy(t, testRootKey(t))
	keyID, algorithm, wrapped, err := kh.GenerateProjectDataKey("project-1")
	if err != nil {
		t.Fatalf("GenerateProjectDataKey: %v", err)
	}
	// AAD binds the wrapped key to its project.
	if _, err := kh.UnwrapProjectDataKey("project-2", keyID, algorithm, wrapped); err == nil {
		t.Fatal("data key unwrapped under a different project scope")
	}
}

func TestKeyScopeAADDistinguishesFields(t *testing.T) {
	base := testScope()
	if !bytes.Equal(base.AAD(), base.AAD()) {
		t.Fatal("AAD must be deterministic for the same scope")
	}
	mutated := []struct {
		name  string
		scope secrets.Scope
	}{
		{name: "different project", scope: withField(base, func(s *secrets.Scope) { s.ProjectID = "proj-other" })},
		{name: "different environment", scope: withField(base, func(s *secrets.Scope) { s.EnvironmentID = "env-other" })},
		{name: "different folder", scope: withField(base, func(s *secrets.Scope) { s.FolderID = "fld-other" })},
		{name: "different secret", scope: withField(base, func(s *secrets.Scope) { s.SecretID = "sec-other" })},
		{name: "different version", scope: withField(base, func(s *secrets.Scope) { s.Version = 2 })},
	}
	for _, tt := range mutated {
		t.Run(tt.name, func(t *testing.T) {
			if bytes.Equal(base.AAD(), tt.scope.AAD()) {
				t.Fatalf("AAD did not change for %s", tt.name)
			}
		})
	}
}

func withField(s secrets.Scope, mutate func(*secrets.Scope)) secrets.Scope {
	mutate(&s)
	return s
}

func TestSealOpenRoundTrip(t *testing.T) {
	kh := mustHierarchy(t, testRootKey(t))
	scope := testScope()
	for _, pt := range []string{"", "x", "postgres://user:pass@host/db", "ünïcödé-π", "line1\nline2"} {
		nonce, ciphertext, err := kh.Seal(bytes.Repeat([]byte{7}, 32), scope, pt)
		if err != nil {
			t.Fatalf("Seal(%q): %v", pt, err)
		}
		if nonce == "" || ciphertext == "" {
			t.Fatalf("empty envelope for %q", pt)
		}
		if pt != "" && ciphertext == pt {
			t.Fatalf("ciphertext equals plaintext for %q", pt)
		}
		got, err := kh.Open(bytes.Repeat([]byte{7}, 32), scope, nonce, ciphertext)
		if err != nil {
			t.Fatalf("Open(%q): %v", pt, err)
		}
		if got != pt {
			t.Fatalf("round trip mismatch: got %q want %q", got, pt)
		}
	}
}

func TestSealUsesFreshNoncePerCall(t *testing.T) {
	kh := mustHierarchy(t, testRootKey(t))
	dataKey := bytes.Repeat([]byte{9}, 32)
	scope := testScope()
	nonce1, ct1, err := kh.Seal(dataKey, scope, "same-plaintext")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	nonce2, ct2, err := kh.Seal(dataKey, scope, "same-plaintext")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if nonce1 == nonce2 {
		t.Fatal("nonces must be unique per seal")
	}
	if ct1 == ct2 {
		t.Fatal("ciphertexts must differ when nonces differ")
	}
}

func TestOpenWrongScopeFails(t *testing.T) {
	kh := mustHierarchy(t, testRootKey(t))
	dataKey := bytes.Repeat([]byte{9}, 32)
	scopeA := testScope()
	scopeB := testScope()
	scopeB.FolderID = "fld-00000000000000000000000000000009"
	nonce, ciphertext, err := kh.Seal(dataKey, scopeA, "scoped-value")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := kh.Open(dataKey, scopeB, nonce, ciphertext); err == nil {
		t.Fatal("ciphertext decrypted under a different scope")
	}
	if _, err := kh.Open(dataKey, scopeA, nonce, ciphertext); err != nil {
		t.Fatalf("same-scope open failed: %v", err)
	}
}

func TestOpenTamperedNonceOrCiphertextFails(t *testing.T) {
	kh := mustHierarchy(t, testRootKey(t))
	dataKey := bytes.Repeat([]byte{9}, 32)
	scope := testScope()
	nonce, ciphertext, err := kh.Seal(dataKey, scope, "tamper-me")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	nonceBytes := mustDecodeB64(t, nonce)
	ctBytes := mustDecodeB64(t, ciphertext)
	nonceBytes[0] ^= 0xff
	ctBytes[len(ctBytes)-1] ^= 0xff
	if _, err := kh.Open(dataKey, scope, base64.StdEncoding.EncodeToString(nonceBytes), ciphertext); err == nil {
		t.Fatal("tampered nonce accepted")
	}
	if _, err := kh.Open(dataKey, scope, nonce, base64.StdEncoding.EncodeToString(ctBytes)); err == nil {
		t.Fatal("tampered ciphertext accepted")
	}
	if _, err := kh.Open(dataKey, scope, "not-base64!!", ciphertext); err == nil {
		t.Fatal("malformed nonce accepted")
	}
}

func TestVersionSealOpenWrongVersionFails(t *testing.T) {
	kh := mustHierarchy(t, testRootKey(t))
	dataKey := bytes.Repeat([]byte{9}, 32)
	v1 := testScope()
	v2 := testScope()
	v2.Version = 2
	nonce, ciphertext, err := kh.Seal(dataKey, v1, "v1-value")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if _, err := kh.Open(dataKey, v2, nonce, ciphertext); err == nil {
		t.Fatal("version 1 ciphertext opened as version 2")
	}
}

func mustDecodeB64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("decode base64: %v", err)
	}
	return b
}
