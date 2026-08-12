package deploy

import (
	"context"
	"fmt"
	"strings"

	"github.com/danielvm/bigbase/components/internal/envcrypto"
)

// FetchSiteEnvVars queries site_env_vars for the given site and returns
// decrypted KEY=value strings. When buildTime is true, only is_build_time=1
// vars are returned; when false, only is_runtime=1 vars are returned.
// Returns nil (no error) when the table does not exist yet.
func (d *Deploy) FetchSiteEnvVars(ctx context.Context, siteID string, buildTime bool) ([]string, error) {
	if siteID == "" {
		return nil, nil
	}

	col := "is_runtime"
	if buildTime {
		col = "is_build_time"
	}

	rows, err := d.db.QueryContext(ctx,
		"SELECT key, value_encrypted FROM site_env_vars WHERE site_id = ? AND "+col+" = 1",
		siteID)
	if err != nil {
		// Table may not exist in older deployments — treat as empty.
		if isNoSuchTable(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch site env vars: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []string
	for rows.Next() {
		var key, encrypted string
		if err := rows.Scan(&key, &encrypted); err != nil {
			return nil, fmt.Errorf("fetch site env vars")
		}
		value, err := envcrypto.Decrypt(d.envKey, encrypted)
		if err != nil {
			return nil, fmt.Errorf("fetch site env vars")
		}
		out = append(out, key+"="+value)
	}
	// F05: check rows.Err() after iteration
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fetch site env vars iteration: %w", err)
	}
	return out, nil
}

// EncryptEnvValue encrypts a plaintext value using the deploy component's
// configured encryption key. Exported for use in tests. Returns plaintext
// unchanged when no key is configured.
func (d *Deploy) EncryptEnvValue(plaintext string) (string, error) {
	return envcrypto.Encrypt(d.envKey, plaintext)
}

func isNoSuchTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such table") || strings.Contains(msg, "does not exist")
}

// parseEnvEncryptionKey validates and decodes the hex env encryption key.
// Returns nil, nil when empty (encryption disabled).
func parseEnvEncryptionKey(keyHex string) ([]byte, error) {
	return envcrypto.ParseKey(keyHex)
}
