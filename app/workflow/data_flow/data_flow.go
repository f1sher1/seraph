package data_flow

import (
	"seraph/app/expressions"
	"seraph/app/objects"
	"seraph/pkg/log"
)

type DataContext map[string]interface{}

func NewDataContext(data ...map[string]interface{}) DataContext {
	ctx := DataContext{}
	for _, d := range data {
		for name, value := range d {
			ctx[name] = value
		}
	}
	return ctx
}

func GetTaskInfo(taskExec *objects.TaskExecution) map[string]interface{} {
	return map[string]interface{}{
		"__task_execution": map[string]interface{}{
			"id":   taskExec.Execution.ID,
			"name": taskExec.Execution.Name,
		},
	}
}

func GetWorkflowEnvironment(wfEx *objects.WorkflowExecution) objects.Table {
	environ := objects.Table{
		"__env": map[string]interface{}{},
	}
	if wfEx == nil {
		return environ
	}

	env, ok := wfEx.Params["env"]
	if ok {
		environ["__env"] = env
	}
	return environ
}

func EvaluateTaskOutBondContext(taskEx *objects.TaskExecution) objects.Table {
	ctx := objects.Table{}
	for k, v := range taskEx.InContext {
		ctx[k] = v
	}
	for k, v := range taskEx.Published {
		ctx[k] = v
	}
	return ctx
}

func getCurrentTaskMap(taskEx *objects.TaskExecution) map[string]interface{} {
	return map[string]interface{}{
		"__task_execution": map[string]string{
			"id":   taskEx.ID,
			"name": taskEx.Name,
		},
	}
}

func PublishVariables(taskEx *objects.TaskExecution, taskSpec *objects.TaskSpec) {
	wfEx, err := taskEx.GetWorkflowExecution()
	if err != nil {
		log.Errorf(taskEx.GetContext(), "publish variables on task execution %s failed, error: %s", taskEx.ID, err.Error())
		return
	}
	exprCtx := NewDataContext(
		getCurrentTaskMap(taskEx),
		taskEx.InContext,
		GetWorkflowEnvironment(wfEx),
		wfEx.Context,
		wfEx.Input,
	)
	if _, ok := exprCtx[taskEx.Name]; ok {
		log.Warnf(taskEx.GetContext(), "Shadowing context variable with task name while publishing: %s", taskEx.Name)
	}

	publishSpec := taskSpec.GetPublish(taskEx.State)
	if publishSpec == nil {
		return
	}

	branchVars := publishSpec.Branch
	taskPublished, err := expressions.EvaluateRecursively(branchVars, exprCtx)
	if err != nil {
		return
	}
	taskEx.Published = taskPublished.(map[string]interface{})

	// todo; add global variables
}
