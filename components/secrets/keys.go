package secrets

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"

	"github.com/danielvm/bigbase/components/internal/envcrypto"
)

// AlgorithmAES256GCM is the only supported envelope algorithm in this release.
// Versions and key records carry it explicitly so unsupported or tampered
// algorithm identifiers fail closed instead of being guessed.
const AlgorithmAES256GCM = "aes-256-gcm"

// Key record states. Exactly one 'active' key exists per project (enforced by
// a partial unique index); 'rotating' records are the rotation checkpoint and
// are promoted to 'active' by CompleteKeyRotation; 'retired' keys remain
// stored so old versions stay decryptable.
const (
	KeyStateActive   = "active"
	KeyStateRotating = "rotating"
	KeyStateRetired  = "retired"
)

// rootKeyLen is the canonical root key size in bytes (AES-256).
const rootKeyLen = 32

// Scope identifies the stable location of a secret version. Every field is
// authenticated as AAD, so ciphertext copied to another scope fails decryption.
type Scope struct {
	ProjectID     string
	EnvironmentID string
	FolderID      string
	SecretID      string
	Version       int
}

// AAD returns the canonical unambiguous encoding of the scope: length-prefixed
// identifiers followed by the version. Length prefixes make the encoding
// injective; IDs are hex and the version is decimal, so no field can smuggle a
// separator into another field.
func (s Scope) AAD() []byte {
	var b bytes.Buffer
	for _, id := range []string{s.ProjectID, s.EnvironmentID, s.FolderID, s.SecretID} {
		b.WriteString(strconv.Itoa(len(id)))
		b.WriteByte(':')
		b.WriteString(id)
	}
	b.WriteString("v:")
	b.WriteString(strconv.Itoa(s.Version))
	return b.Bytes()
}

// keyRecordAAD binds a wrapped project data key to its project, so a key
// record copied to another project cannot be unwrapped.
func keyRecordAAD(projectID string) []byte {
	var b bytes.Buffer
	b.WriteString("bigbase:key-record:")
	b.WriteString(strconv.Itoa(len(projectID)))
	b.WriteByte(':')
	b.WriteString(projectID)
	return b.Bytes()
}

// KeyHierarchy owns root-key bootstrap, per-project data keys, and versioned
// AES-256-GCM envelopes with scope-bound authenticated data. It never logs key
// material; wrapped keys and ciphertext are the only bytes that persist.
type KeyHierarchy struct {
	rootKey []byte
}

// NewKeyHierarchy validates the canonical 32-byte root key. An invalid key is
// a configuration error: callers must fail closed rather than fall back to
// plaintext.
func NewKeyHierarchy(rootKey []byte) (*KeyHierarchy, error) {
	if len(rootKey) != rootKeyLen {
		return nil, fmt.Errorf("root key must be %d bytes", rootKeyLen)
	}
	return &KeyHierarchy{rootKey: rootKey}, nil
}

// ParseRootKey decodes the canonical base64-encoded 32-byte root key. It is
// the composition-root seam for the secrets component and consumes the
// envcrypto primitive contract (s01).
func ParseRootKey(raw string) ([]byte, error) {
	return envcrypto.ParseRootKey(raw)
}

// GenerateProjectDataKey creates a fresh random 32-byte data key, wraps it
// under the root key bound to projectID, and returns the identifiers and
// wrapped material for a project_key_records row. The plaintext data key
// never leaves this function.
func (kh *KeyHierarchy) GenerateProjectDataKey(projectID string) (keyID, algorithm, encryptedKey string, err error) {
	dataKey := make([]byte, rootKeyLen)
	if _, err := io.ReadFull(rand.Reader, dataKey); err != nil {
		return "", "", "", fmt.Errorf("generate data key: %w", err)
	}
	keyID, err = generateKeyID()
	if err != nil {
		return "", "", "", err
	}
	nonce, sealed, err := sealAESGCM(kh.rootKey, keyRecordAAD(projectID), dataKey)
	if err != nil {
		return "", "", "", fmt.Errorf("wrap data key: %w", err)
	}
	blob := make([]byte, 0, len(nonce)+len(sealed))
	blob = append(blob, nonce...)
	blob = append(blob, sealed...)
	return keyID, AlgorithmAES256GCM, base64.StdEncoding.EncodeToString(blob), nil
}

// UnwrapProjectDataKey decrypts and validates a stored data key record. It
// rejects unsupported algorithms, tampered blobs, wrong root keys, and
// malformed key lengths.
func (kh *KeyHierarchy) UnwrapProjectDataKey(projectID, keyID, algorithm, encryptedKey string) ([]byte, error) {
	if algorithm != AlgorithmAES256GCM {
		return nil, fmt.Errorf("unsupported algorithm %q", algorithm)
	}
	blob, err := base64.StdEncoding.DecodeString(encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("decode wrapped key: %w", err)
	}
	dataKey, err := openAESGCM(kh.rootKey, keyRecordAAD(projectID), blob)
	if err != nil {
		return nil, fmt.Errorf("unwrap project data key %q: %w", keyID, err)
	}
	if len(dataKey) != rootKeyLen {
		return nil, fmt.Errorf("unwrapped data key has invalid length %d", len(dataKey))
	}
	return dataKey, nil
}

// Seal encrypts plaintext under dataKey with a fresh nonce and scope-bound
// AAD. Nonce and ciphertext are returned separately so rows store them in
// distinct ciphertext-only columns.
func (kh *KeyHierarchy) Seal(dataKey []byte, scope Scope, plaintext string) (nonce, ciphertext string, err error) {
	nonceBytes, sealed, err := sealAESGCM(dataKey, scope.AAD(), []byte(plaintext))
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(nonceBytes), base64.StdEncoding.EncodeToString(sealed), nil
}

// Open decrypts a version blob, authenticating the scope AAD, nonce, and
// ciphertext. Any tampering fails closed with an error and no plaintext.
func (kh *KeyHierarchy) Open(dataKey []byte, scope Scope, nonce, ciphertext string) (string, error) {
	nonceBytes, err := base64.StdEncoding.DecodeString(nonce)
	if err != nil {
		return "", fmt.Errorf("decode nonce: %w", err)
	}
	sealed, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decode ciphertext: %w", err)
	}
	blob := make([]byte, 0, len(nonceBytes)+len(sealed))
	blob = append(blob, nonceBytes...)
	blob = append(blob, sealed...)
	plaintext, err := openAESGCM(dataKey, scope.AAD(), blob)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func sealAESGCM(key, aad, plaintext []byte) (nonce, sealed []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("new gcm: %w", err)
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}
	sealed = gcm.Seal(nil, nonce, plaintext, aad)
	return nonce, sealed, nil
}

func openAESGCM(key, aad []byte, blob []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	if len(blob) < gcm.NonceSize()+gcm.Overhead() {
		return nil, fmt.Errorf("blob too short")
	}
	plaintext, err := gcm.Open(nil, blob[:gcm.NonceSize()], blob[gcm.NonceSize():], aad)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return plaintext, nil
}

func generateKeyID() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", fmt.Errorf("generate key id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
