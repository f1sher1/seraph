package states

var (
	// IDLE Task is not started yet.
	IDLE = "IDLE"

	// WAITING Task execution object has been created, but it is not ready to start because
	// some preconditions are not met.
	// NOTE:
	// The task may never run just because some preconditions may never be met.
	WAITING = "WAITING"

	// RUNNING Task, action or workflow is currently being executed.
	RUNNING = "RUNNING"

	// RUNNING_DELAYED Task is in the running state but temporarily delayed.
	RUNNING_DELAYED = "DELAYED"

	// PAUSED Task, action or workflow has been paused.
	PAUSED = "PAUSED"

	// SUCCESS Task, action or workflow has finished successfully.
	SUCCESS = "SUCCESS"

	// CANCELLED Task, action or workflow has been cancelled.
	CANCELLED = "CANCELLED"

	// ERROR Task, action or workflow has finished with an error.
	ERROR = "ERROR"

	_ALL = []string{
		IDLE,
		WAITING,
		RUNNING,
		RUNNING_DELAYED,
		PAUSED,
		SUCCESS,
		CANCELLED,
		ERROR,
	}

	CompleteStates = []string {
		SUCCESS, ERROR, CANCELLED,
	}

	IncompleteStates = []string {
		IDLE, RUNNING, WAITING, RUNNING_DELAYED, PAUSED,
	}
)

func IsCompleted(state string) bool {
	for _, s := range CompleteStates {
		if s == state {
			return true
		}
	}
	return false
}

func IsRunning(state string) bool {
	return state == RUNNING
}

func IsSuccess(state string) bool {
	return state == SUCCESS
}

func IsCanceled(state string) bool {
	return state == CANCELLED
}

func IsPaused(state string) bool {
	return state == PAUSED
}

func IsErrored(state string) bool {
	return state == ERROR
}

func IsWaiting(state string) bool {
	return state == WAITING
}

func IsDelayed(state string) bool {
	return state == RUNNING_DELAYED
}

func ValidateStateTransition(curState, state string) error {
	return nil
}

type LogicState struct {
	State string
	StateInfo   string
	TriggeredBy []map[string]interface{}
	Cardinality int
}

