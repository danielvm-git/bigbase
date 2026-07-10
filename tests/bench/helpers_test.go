package bench

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"testing"
)

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
