package functions

import (
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
		defer resp.Body.Close()

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
