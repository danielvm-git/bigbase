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
	Execute(source string, timeout int) (*RunOutput, *RunError)
}

var runtimes = map[string]Runtime{
	"javascript": &jsRuntime{},
}

type jsRuntime struct{}

func (*jsRuntime) Execute(source string, timeout int) (*RunOutput, *RunError) {
	vm := goja.New()

	var mu sync.Mutex
	logs := make([]string, 0)

	_ = vm.Set("console", map[string]any{
		"log": func(msg string) {
			mu.Lock()
			logs = append(logs, msg)
			mu.Unlock()
		},
		"info": func(msg string) {
			mu.Lock()
			logs = append(logs, "[info] "+msg)
			mu.Unlock()
		},
		"warn": func(msg string) {
			mu.Lock()
			logs = append(logs, "[warn] "+msg)
			mu.Unlock()
		},
		"error": func(msg string) {
			mu.Lock()
			logs = append(logs, "[error] "+msg)
			mu.Unlock()
		},
	})

	timeoutDur := time.Duration(timeout) * time.Second
	if timeout <= 0 {
		timeoutDur = 30 * time.Second
	}

	type result struct {
		val goja.Value
		err error
	}

	done := make(chan result, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- result{err: fmt.Errorf("panic: %v", r)}
			}
		}()

		val, err := vm.RunString("(function() {" + source + "\n})()")
		done <- result{val: val, err: err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			return &RunOutput{Logs: logs}, &RunError{Message: r.err.Error()}
		}
		return &RunOutput{Logs: logs, Result: r.val.Export()}, nil
	case <-time.After(timeoutDur):
		return &RunOutput{Logs: logs}, &RunError{Message: fmt.Sprintf("execution timed out after %d seconds", timeout)}
	}
}
