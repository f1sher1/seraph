package interfaces

import (
	"seraph/app/objects"
	"time"
)

type Task interface {
	SetUniqueKey(string)
	SetWait(bool)
	Run() error
	ToMap() map[string]interface{}
	ForceFail(error)
	Complete(state string, stateInfo string) error
	OnActionComplete(ex interface{}) error
	GetTaskExecution() *objects.TaskExecution
	SetState(state string, stateInfo string, processed interface{}) error
	Notify(oldState string, newState string)

	GetTriggerEvent() string
	GetTaskSpec() *objects.TaskSpec
	HandledErrors() bool
}

type TaskAction interface {
	Initialize() error
	ValidateInput(input objects.Table) error
	Schedule(input objects.Table, target string, timeout time.Duration, index int, description string) error

	Complete(result *objects.ActionResult) error
	Fail(msg string)
	GetTaskExecution() *objects.TaskExecution
}
