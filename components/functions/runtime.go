package functions

import (
	"fmt"
	"sync"
	"time"

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

type Runtime interface {
	Execute(source string, timeout int) (*RunOutput, error)
}

type jsRuntime struct{}

func (*jsRuntime) Execute(source string, timeout int) (*RunOutput, error) {
	vm := goja.New()
	col := injectConsole(vm)

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
