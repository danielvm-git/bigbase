package github_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/danielvm/bigbase/components/github"
)

func TestVerifyWebhookSignatureValid(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"ref":"refs/heads/main"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !github.VerifyWebhookSignatureForTest(secret, body, sig) {
		t.Fatal("expected valid signature")
	}
}

func TestVerifyWebhookSignatureInvalid(t *testing.T) {
	if github.VerifyWebhookSignatureForTest("secret", []byte("x"), "sha256=deadbeef") {
		t.Fatal("expected invalid signature")
	}
}
