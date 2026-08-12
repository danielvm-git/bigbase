// Package secrets implements the native Infisical-inspired project secret
// manager (e89): SecretFolder/Secret/SecretVersion metadata, a versioned
// AES-256-GCM envelope encryption KeyHierarchy, and the typed SecretManager
// public seam consumed by the REST, Deploy, and MCP adapters.
//
// Storage is ciphertext-only. List and mutation results are metadata-only;
// plaintext is available exclusively through the explicit value-read methods.
package secrets

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/danielvm/bigbase/kernel"
)

const version = "0.1.0"

// DBer is an alias for kernel.DBer.
type DBer = kernel.DBer

// Options configures the Secrets component.
// RootKey is the canonical base64-decoded 32-byte root key; it must be
// validated by the composition root (see ParseRootKey) and never logged.
type Options struct {
	DB      DBer
	Logger  kernel.Logger
	RootKey []byte
}

// Secrets is the SecretManager component. It implements kernel.Component and
// the typed SecretManager seam on the same type, so adapters receive it
// directly through composition-root injection.
type Secrets struct {
	db     DBer
	logger kernel.Logger
	kh     *KeyHierarchy
}

var _ kernel.Component = (*Secrets)(nil)
var _ SecretManager = (*Secrets)(nil)

// New validates configuration and constructs the component. A missing or
// invalid root key is a hard error: production must fail closed rather than
// store project secrets in plaintext.
func New(opts Options) (*Secrets, error) {
	logger := opts.Logger
	if logger == nil {
		logger = kernel.NoopLogger{}
	}
	if opts.DB == nil {
		return nil, errors.New("secrets: DB is required")
	}
	kh, err := NewKeyHierarchy(opts.RootKey)
	if err != nil {
		return nil, fmt.Errorf("secrets: invalid root key: %w", err)
	}
	return &Secrets{db: opts.DB, logger: logger, kh: kh}, nil
}

func (s *Secrets) Name() string                                    { return "secrets" }
func (s *Secrets) Version() string                                 { return version }
func (s *Secrets) Dependencies() []string                          { return []string{"projects"} }
func (s *Secrets) Init(_ *kernel.Context, _ json.RawMessage) error { return nil }

// Start applies the secret schema. The kernel guarantees projects starts first
// (Dependencies), so the projects and project_environments tables exist before
// these foreign keys are created.
func (s *Secrets) Start(_ *kernel.Context) error {
	if s.db == nil {
		return errors.New("secrets: DB is required")
	}
	for _, m := range schemaMigrations {
		if err := s.db.Migrate(m); err != nil {
			return fmt.Errorf("secrets migrate: %w", err)
		}
	}
	for _, m := range indexMigrations {
		if err := s.db.Migrate(m); err != nil {
			return fmt.Errorf("secrets migrate index: %w", err)
		}
	}
	return nil
}

func (s *Secrets) Stop(_ *kernel.Context) error { return nil }
