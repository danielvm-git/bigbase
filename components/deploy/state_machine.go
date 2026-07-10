package deploy

// DeploymentState represents the lifecycle stage of a deployment.
type DeploymentState string

const (
	StatePending    DeploymentState = "pending"
	StateBuilding   DeploymentState = "building"
	StateDeploying  DeploymentState = "deploying"
	StateRunning    DeploymentState = "running"
	StateDraining   DeploymentState = "draining"
	StateStopped    DeploymentState = "stopped"
	StateFailed     DeploymentState = "failed"
	StateRolledBack DeploymentState = "rolled_back"
)

// StatusTransition records a single state change in the deployment lifecycle.
type StatusTransition struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Timestamp string `json:"timestamp"`
}

// StateMachine defines valid transitions between deployment states.
// 'running' is the terminal state (kept to avoid changing 5 SQL queries
// that all use WHERE status = 'running'). 'deploying' is the only new,
// transitional state (building → deploying → running for process apps).
type StateMachine struct {
	valid map[string][]string
}

func NewStateMachine() *StateMachine {
	return &StateMachine{
		valid: map[string][]string{
			string(StatePending):    {string(StateBuilding)},
			string(StateBuilding):   {string(StateDeploying), string(StateRunning), string(StateFailed)},
			string(StateDeploying):  {string(StateRunning), string(StateFailed)},
			string(StateRunning):    {string(StateFailed), string(StateRolledBack), string(StateDraining)},
			string(StateDraining):   {string(StateStopped), string(StateFailed), string(StateRunning)},
			string(StateStopped):    {string(StateRunning)},
			string(StateFailed):     {},
			string(StateRolledBack): {},
		},
	}
}

// CanTransition reports whether moving from one state to another is valid.
func (sm *StateMachine) CanTransition(from, to string) bool {
	if from == to {
		return false // idempotent transitions are handled by TransitionState, not the machine
	}
	next, ok := sm.valid[from]
	if !ok {
		return false
	}
	for _, n := range next {
		if n == to {
			return true
		}
	}
	return false
}

// ValidTransitions returns all states reachable from the given state.
func (sm *StateMachine) ValidTransitions(from string) []string {
	return sm.valid[from]
}

// IsValidState reports whether the given string is a known DeploymentState.
func (sm *StateMachine) IsValidState(status string) bool {
	_, ok := sm.valid[status]
	return ok
}
