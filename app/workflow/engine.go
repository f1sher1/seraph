package workflow

import (
	"seraph/app/objects"
	"seraph/app/workflow/interfaces"
	"seraph/pkg/contextx"
)

type NewEngineTaskFunc func(ctx *contextx.Context, wfEx *objects.WorkflowExecution, wfSpec *objects.WorkflowSpec, taskSpec *objects.TaskSpec, taskCtx map[string]interface{}) interfaces.Task

var (
	engineTasks = map[string]NewEngineTaskFunc{
		"noop":    NewEngineNoopCommand,
		"fail":    NewEngineFailCommand,
		"succeed": NewEngineSuccessCommand,
		"pause":   NewEnginePauseCommand,
	}
)

func NewEnginePauseCommand(ctx *contextx.Context, wfEx *objects.WorkflowExecution, wfSpec *objects.WorkflowSpec, taskSpec *objects.TaskSpec, taskCtx map[string]interface{}) interfaces.Task {
	return &EnginePauseTask{}
}

func NewEngineSuccessCommand(ctx *contextx.Context, wfEx *objects.WorkflowExecution, wfSpec *objects.WorkflowSpec, taskSpec *objects.TaskSpec, taskCtx map[string]interface{}) interfaces.Task {
	return &EngineSuccessTask{}
}

func NewEngineFailCommand(ctx *contextx.Context, wfEx *objects.WorkflowExecution, wfSpec *objects.WorkflowSpec, taskSpec *objects.TaskSpec, taskCtx map[string]interface{}) interfaces.Task {
	return &EngineFailTask{}
}

func NewEngineNoopCommand(ctx *contextx.Context, wfEx *objects.WorkflowExecution, wfSpec *objects.WorkflowSpec, taskSpec *objects.TaskSpec, taskCtx map[string]interface{}) interfaces.Task {
	return &EngineNoopTask{}
}

type EngineNoopTask struct {
	BaseTask
}

type EngineFailTask struct {
	BaseTask
}

type EngineSuccessTask struct {
	BaseTask
}

type EnginePauseTask struct {
	BaseTask
}

func IsEngineTask(name string) bool {
	_, ok := engineTasks[name]
	return ok
}

func GetEngineTask(name string) NewEngineTaskFunc {
	newFunc, ok := engineTasks[name]
	if ok {
		return newFunc
	}
	return nil
}
