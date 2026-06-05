package bench

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"
)

type noopLogger struct{}

func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Warn(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}
func (noopLogger) Debug(msg string, args ...any) {}

func decodeJSON(tb testing.TB, res *http.Response, v any) {
	tb.Helper()
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		tb.Fatalf("read body: %v", err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		tb.Fatalf("json decode: %v\nbody: %s", err, string(body))
	}
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
