package functions

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
	return &RunOutput{Logs: []string{}, Result: nil}, &RunError{Message: "goja runtime not yet available"}
}
