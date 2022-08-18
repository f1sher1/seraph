package interfaces

import (
	"seraph/app/objects"
	"seraph/app/workflow/states"
)

type WorkflowController interface {
	Initialize(*objects.WorkflowSpec) error
	SetWorkflowExecution(wfEx *objects.WorkflowExecution)
	GetNextTasks(taskExec *objects.TaskExecution) ([]Task, error)
	GetStartTasks() ([]Task, error)
	FindIndirectlyAffectedTaskExecutions(name string) []*objects.TaskExecution
	GetLogicTaskState(ex *objects.TaskExecution) *states.LogicState
	EvaluateWorkflowFinalContext() (map[string]interface{}, error)
	GetNextTasksForTask(taskEx *objects.TaskExecution) ([]Task, error)
	MayCompleteWorkflow(ex *objects.TaskExecution) bool
	AnyCancels() bool
	AllErrorHandled() bool
	IsErrorHandled(t *objects.TaskExecution) bool
}
