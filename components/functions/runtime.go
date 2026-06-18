package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/danielvm/bigbase/kernel"
	"github.com/dop251/goja"
)

type RunOutput struct {
	Logs   []string `json:"logs"`
	Result any      `json:"result"`
}

type RunError struct {
	Message string `json:"message"`
}

func (e *RunError) Error() string { return e.Message }

// RunContext holds per-execution configuration for function runtime injection.
type RunContext struct {
	Env     map[string]string
	DB      kernel.DBer
	Request *http.Request
}

type Runtime interface {
	Execute(source string, timeout int, ctx RunContext) (*RunOutput, error)
}

type jsRuntime struct{}

func (*jsRuntime) Execute(source string, timeout int, ctx RunContext) (*RunOutput, error) {
	vm := goja.New()
	col := injectConsole(vm)
	injectEnv(vm, ctx.Env)
	injectFetch(vm, ctx.Env)
	injectDB(vm, ctx.DB)
	injectRequest(vm, ctx.Request)

	code := "(function() {" + source + "\n})()"
	done := execAsync(vm, code)

	timer := time.NewTimer(timeoutDuration(timeout))
	defer timer.Stop()

	select {
	case r := <-done:
		if r.err != nil {
			return &RunOutput{Logs: col.snapshot()}, &RunError{Message: r.err.Error()}
		}
		return &RunOutput{Logs: col.snapshot(), Result: r.val.Export()}, nil
	case <-timer.C:
		vm.Interrupt("execution timeout")
		<-done
		return &RunOutput{Logs: col.snapshot()}, &RunError{Message: fmt.Sprintf("execution timed out after %d seconds", timeout)}
	}
}

// injectEnv exposes the Function's env map as a read-only global `env` object.
func injectEnv(vm *goja.Runtime, env map[string]string) {
	obj := vm.NewObject()
	for k, v := range env {
		val := v // capture
		_ = obj.Set(k, val)
	}
	_ = vm.Set("env", obj)
}

// injectFetch injects a synchronous `fetch(url)` global function.
// The allowlist is read from env["ALLOWED_HOSTS"] (comma-separated hosts).
// If no allowlist is set, only localhost is allowed.
func injectFetch(vm *goja.Runtime, env map[string]string) {
	allowed := allowedHosts(env)

	_ = vm.Set("fetch", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("fetch requires a URL argument")))
		}
		rawURL := call.Arguments[0].String()

		parsed, err := url.Parse(rawURL)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("fetch: invalid URL %q: %w", rawURL, err)))
		}

		if !isHostAllowed(parsed.Host, allowed) {
			panic(vm.NewGoError(fmt.Errorf("fetch: host %q not in allowlist", parsed.Host)))
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(rawURL)
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("fetch: request failed: %w", err)))
		}
		defer func() { _ = resp.Body.Close() }()

		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MB limit
		if err != nil {
			panic(vm.NewGoError(fmt.Errorf("fetch: read body failed: %w", err)))
		}

		result := vm.NewObject()
		_ = result.Set("status", resp.StatusCode)

		headers := vm.NewObject()
		for k, vals := range resp.Header {
			_ = headers.Set(k, strings.Join(vals, ", "))
		}
		_ = result.Set("headers", headers)
		_ = result.Set("body", string(bodyBytes))

		return result
	})
}

// allowedHosts parses the ALLOWED_HOSTS env var. Default: localhost only.
func allowedHosts(env map[string]string) []string {
	raw, ok := env["ALLOWED_HOSTS"]
	if !ok || raw == "" {
		return []string{"localhost", "127.0.0.1"}
	}
	hosts := strings.Split(raw, ",")
	for i, h := range hosts {
		hosts[i] = strings.TrimSpace(h)
	}
	return hosts
}

// isHostAllowed checks if host matches any entry in the allowlist.
func isHostAllowed(host string, allowed []string) bool {
	// Strip port for comparison
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}
	for _, a := range allowed {
		if a == hostname || a == host {
			return true
		}
	}
	return false
}

// injectDB injects a `db` global with collection() methods for CRUD operations.
// Collection names are sanitized to prevent SQL injection.
func injectDB(vm *goja.Runtime, dber kernel.DBer) {
	dbObj := vm.NewObject()
	_ = dbObj.Set("collection", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.NewGoError(fmt.Errorf("db.collection requires a name argument")))
		}
		name := call.Arguments[0].String()
		if err := validateCollectionName(name); err != nil {
			panic(vm.NewGoError(err))
		}

		// Ensure the collection table exists
		ctx := context.Background()
		createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INTEGER PRIMARY KEY AUTOINCREMENT, data TEXT)", name)
		if err := dber.Migrate(createSQL); err != nil {
			panic(vm.NewGoError(fmt.Errorf("db: failed to create collection %q: %w", name, err)))
		}

		col := vm.NewObject()

		// create(data) → inserts a JSON record
		_ = col.Set("create", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) < 1 {
				panic(vm.NewGoError(fmt.Errorf("create requires a data argument")))
			}
			data := call.Arguments[0].Export()
			dataJSON, err := json.Marshal(data)
			if err != nil {
				panic(vm.NewGoError(fmt.Errorf("create: marshal failed: %w", err)))
			}
			result, err := dber.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s (data) VALUES (?)", name), string(dataJSON))
			if err != nil {
				panic(vm.NewGoError(fmt.Errorf("create: insert failed: %w", err)))
			}
			id, _ := result.LastInsertId()
			out := vm.NewObject()
			_ = out.Set("id", id)
			return out
		})

		// list() → returns all records
		_ = col.Set("list", func(call goja.FunctionCall) goja.Value {
			rows, err := dber.QueryContext(ctx, fmt.Sprintf("SELECT id, data FROM %s ORDER BY id", name))
			if err != nil {
				panic(vm.NewGoError(fmt.Errorf("list: query failed: %w", err)))
			}
			defer func() { _ = rows.Close() }()

			var items []map[string]any
			for rows.Next() {
				var id int64
				var dataStr string
				if err := rows.Scan(&id, &dataStr); err != nil {
					continue
				}
				record := map[string]any{"id": id}
				var data map[string]any
				if json.Unmarshal([]byte(dataStr), &data) == nil {
					for k, v := range data {
						record[k] = v
					}
				}
				items = append(items, record)
			}
			return vm.ToValue(items)
		})

		// get(id) → returns single record or null
		_ = col.Set("get", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) < 1 {
				panic(vm.NewGoError(fmt.Errorf("get requires an id argument")))
			}
			id := call.Arguments[0].ToInteger()
			row := dber.QueryRowContext(ctx, fmt.Sprintf("SELECT id, data FROM %s WHERE id = ?", name), id)
			var rid int64
			var dataStr string
			if err := row.Scan(&rid, &dataStr); err != nil {
				return goja.Null()
			}
			record := map[string]any{"id": rid}
			var data map[string]any
			if json.Unmarshal([]byte(dataStr), &data) == nil {
				for k, v := range data {
					record[k] = v
				}
			}
			return vm.ToValue(record)
		})

		// update(id, data) → merges data into existing record
		_ = col.Set("update", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) < 2 {
				panic(vm.NewGoError(fmt.Errorf("update requires id and data arguments")))
			}
			id := call.Arguments[0].ToInteger()
			// Fetch existing
			row := dber.QueryRowContext(ctx, fmt.Sprintf("SELECT data FROM %s WHERE id = ?", name), id)
			var existingStr string
			if err := row.Scan(&existingStr); err != nil {
				panic(vm.NewGoError(fmt.Errorf("update: record %d not found", id)))
			}
			var existing map[string]any
			_ = json.Unmarshal([]byte(existingStr), &existing)
			if existing == nil {
				existing = map[string]any{}
			}
			// Merge new data
			newData := call.Arguments[1].Export()
			if newMap, ok := newData.(map[string]any); ok {
				for k, v := range newMap {
					existing[k] = v
				}
			}
			merged, _ := json.Marshal(existing)
			_, err := dber.ExecContext(ctx, fmt.Sprintf("UPDATE %s SET data = ? WHERE id = ?", name), string(merged), id)
			if err != nil {
				panic(vm.NewGoError(fmt.Errorf("update: failed: %w", err)))
			}
			out := vm.NewObject()
			_ = out.Set("status", "updated")
			return out
		})

		// delete(id) → removes a record
		_ = col.Set("delete", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) < 1 {
				panic(vm.NewGoError(fmt.Errorf("delete requires an id argument")))
			}
			id := call.Arguments[0].ToInteger()
			_, err := dber.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE id = ?", name), id)
			if err != nil {
				panic(vm.NewGoError(fmt.Errorf("delete: failed: %w", err)))
			}
			out := vm.NewObject()
			_ = out.Set("status", "deleted")
			return out
		})

		return col
	})
	_ = vm.Set("db", dbObj)
}

// injectRequest exposes the HTTP request context as a global `request` object.
// Provides: method, headers, query, body (parsed JSON or raw string).
func injectRequest(vm *goja.Runtime, r *http.Request) {
	reqObj := vm.NewObject()

	if r != nil {
		_ = reqObj.Set("method", r.Method)

		// Headers
		headers := vm.NewObject()
		for k, vals := range r.Header {
			_ = headers.Set(strings.ToLower(k), strings.Join(vals, ", "))
		}
		_ = reqObj.Set("headers", headers)

		// Query params
		query := vm.NewObject()
		for k, vals := range r.URL.Query() {
			if len(vals) == 1 {
				_ = query.Set(k, vals[0])
			} else {
				_ = query.Set(k, vals)
			}
		}
		_ = reqObj.Set("query", query)

		// Body: read and parse JSON if possible, otherwise raw string
		if r.Body != nil {
			bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
			if err == nil && len(bodyBytes) > 0 {
				var jsonBody any
				if json.Unmarshal(bodyBytes, &jsonBody) == nil {
					_ = reqObj.Set("body", vm.ToValue(jsonBody))
				} else {
					_ = reqObj.Set("body", string(bodyBytes))
				}
			}
		}
	}

	_ = vm.Set("request", reqObj)
}

// validateCollectionName checks the collection name is safe (alphanumeric + underscore).
func validateCollectionName(name string) error {
	if name == "" {
		return fmt.Errorf("collection name is required")
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return fmt.Errorf("invalid character %q in collection name", r)
		}
	}
	return nil
}

type logCollector struct {
	mu   sync.Mutex
	logs []string
}

func (c *logCollector) snapshot() []string {
	c.mu.Lock()
	out := make([]string, len(c.logs))
	copy(out, c.logs)
	c.mu.Unlock()
	return out
}

func (c *logCollector) add(msg string) {
	c.mu.Lock()
	c.logs = append(c.logs, msg)
	c.mu.Unlock()
}

func (c *logCollector) addPrefixed(prefix, msg string) {
	c.mu.Lock()
	c.logs = append(c.logs, prefix+msg)
	c.mu.Unlock()
}

func injectConsole(vm *goja.Runtime) *logCollector {
	col := &logCollector{}

	console := vm.NewObject()
	_ = console.Set("log", func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			col.add(arg.String())
		}
		return goja.Undefined()
	})
	_ = console.Set("info", func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			col.addPrefixed("[info] ", arg.String())
		}
		return goja.Undefined()
	})
	_ = console.Set("warn", func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			col.addPrefixed("[warn] ", arg.String())
		}
		return goja.Undefined()
	})
	_ = console.Set("error", func(call goja.FunctionCall) goja.Value {
		for _, arg := range call.Arguments {
			col.addPrefixed("[error] ", arg.String())
		}
		return goja.Undefined()
	})
	_ = vm.Set("console", console)
	return col
}

type execResult struct {
	val goja.Value
	err error
}

func execAsync(vm *goja.Runtime, code string) chan execResult {
	done := make(chan execResult, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- execResult{err: fmt.Errorf("panic: %v", r)}
			}
		}()
		val, err := vm.RunString(code)
		done <- execResult{val: val, err: err}
	}()
	return done
}

func timeoutDuration(timeout int) time.Duration {
	if timeout <= 0 {
		return 30 * time.Second
	}
	return time.Duration(timeout) * time.Second
}
