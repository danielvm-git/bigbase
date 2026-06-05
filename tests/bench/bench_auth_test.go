package bench

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielvm/bigbase/components/auth"
	"github.com/danielvm/bigbase/components/db"
	"github.com/danielvm/bigbase/kernel"
)

func BenchmarkAuthValidateToken(b *testing.B) {
	logger := noopLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "bench-secret-32-chars!!!"})
	k := kernel.New(logger)
	k.Register(d)
	k.Register(a)
	if err := k.Start(); err != nil {
		b.Fatalf("kernel start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	h := a.Handler()
	body := `{"email":"bench@test.com","password":"BenchPass123!"}`
	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	res := w.Result()
	var regResp struct {
		Token string `json:"token"`
	}
	decodeJSON(b, res, &regResp)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := a.ValidateToken(regResp.Token)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAuthRegister(b *testing.B) {
	logger := noopLogger{}
	d := db.New(db.Options{Path: ":memory:", Logger: logger})
	a := auth.New(auth.Options{DB: d, Logger: logger, Secret: "bench-secret-32-chars!!!"})
	k := kernel.New(logger)
	k.Register(d)
	k.Register(a)
	if err := k.Start(); err != nil {
		b.Fatalf("kernel start: %v", err)
	}
	defer func() { _ = k.Stop() }()

	h := a.Handler()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := `{"email":"bench-` + itoa(i) + `@test.com","password":"BenchPass123!"}`
		req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
	}
}
