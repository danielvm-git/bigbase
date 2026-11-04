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
//
// 'running' is a stable state (kept to avoid changing the 5 SQL queries that
// all use WHERE status = 'running'); the deployment is healthy and serving.
// 'deploying' is a transitional state (building → deploying → running for
// process apps). 'stopped' is a stable terminal state for the build pipeline,
// but it is deliberately revivable via stopped → running: rollback restores a
// previously drained deployment (see components/deploy/rollback.go).
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

// IsActive reports whether the deployment is live or in flight, so a site
// delete must wait for it: pending, building, deploying, draining, running.
func (sm *StateMachine) IsActive(status string) bool {
	return sm.isOneOf(status, StatePending, StateBuilding, StateDeploying, StateDraining, StateRunning)
}

// IsTerminal reports whether the build/start pipeline has finished: running,
// stopped, failed, rolled_back. A terminal deployment still has lifecycle
// transitions (drain, rollback), but no further build work is expected.
func (sm *StateMachine) IsTerminal(status string) bool {
	return sm.isOneOf(status, StateRunning, StateStopped, StateFailed, StateRolledBack)
}

// IsInFlight reports whether the deployment is transient work — a build,
// deploy, or drain currently in progress — that must not be deleted or chosen
// as a rollback target.
func (sm *StateMachine) IsInFlight(status string) bool {
	return sm.isOneOf(status, StatePending, StateBuilding, StateDeploying, StateDraining)
}

// InFlightStatuses returns the statuses for which IsInFlight is true. Callers
// that filter in SQL (rollback target selection) derive their exclusion list
// from here so the sets cannot drift.
func (sm *StateMachine) InFlightStatuses() []string {
	return []string{
		string(StatePending),
		string(StateBuilding),
		string(StateDeploying),
		string(StateDraining),
	}
}

// CanDelete reports whether a deployment may be deleted now. In-flight rows
// must finish first; running is deletable because the delete path stops it
// before removing the row. Unknown/legacy statuses remain deletable.
func (sm *StateMachine) CanDelete(status string) bool {
	return !sm.IsInFlight(status)
}

func (sm *StateMachine) isOneOf(status string, states ...DeploymentState) bool {
	for _, s := range states {
		if status == string(s) {
			return true
		}
	}
	return false
}
