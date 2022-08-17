package workflow

import (
	"fmt"
	"seraph/app/objects"
	"seraph/app/workflow/actions"
	"seraph/app/workflow/interfaces"
	"seraph/pkg/log"
)

func buildActionFromActionExecution(actionEx *objects.ActionExecution) (taskAction interfaces.TaskAction) {
	taskEx, err := actionEx.GetTaskExecution()
	if err != nil {
		log.Errorf(actionEx.GetContext(), "query task execution of action execution %s failed, error: %s", actionEx.TaskExecutionID, err.Error())
	} else if taskEx != nil {
		var actionName string
		var adHocActionFlag bool
		if adHocActionName, ok := actionEx.RuntimeContext["adhoc_action_name"]; ok {
			// nested action
			actionName = adHocActionName.(string)
			adHocActionFlag = true
		} else {
			actionName = actionEx.Name
		}

		actDef, err := objects.QueryActionDefinitionByName(actionEx.GetContext(), actionName)
		if err != nil {
			log.Errorf(actionEx.GetContext(), "query action definition by name %s failed, error: %s", actionName, err.Error())
		} else if actDef != nil {
			if adHocActionFlag {
				taskAction, err = actions.NewAdHocAction(actionEx.GetContext(), actDef, taskEx, nil, nil)
			} else {
				taskAction, err = actions.NewGolangAction(actionEx.GetContext(), actDef, actionEx, taskEx)
			}

			if err != nil {
				log.Errorf(actionEx.GetContext(), "create adhoc action of action execution %s failed, error: %s", actionEx.ID, err.Error())
			} else {
				return
			}
		}
	}
	return nil
}

func buildActionFromWorkflowExecution(wfEx *objects.WorkflowExecution) interfaces.TaskAction {
	taskEx, err := wfEx.GetTaskExecution()
	if err != nil {
		log.Errorf(wfEx.GetContext(), "query task execution of workflow execution %s failed, error: %s", wfEx.TaskExecutionID, err.Error())
	} else if taskEx != nil {
		wfAction, err := actions.NewWorkflowAction(wfEx.Name, taskEx, wfEx)
		if err != nil {
			log.Errorf(wfEx.GetContext(), "create workflow action of workflow execution %s failed, error: %s", wfEx.ID, err.Error())
		} else {
			return wfAction
		}
	}
	return nil
}

func onCommonActionComplete(act interfaces.TaskAction, result *objects.ActionResult, exID string, ex interface{}) {
	taskEx := act.GetTaskExecution()
	err := act.Complete(result)
	if err == nil {
		if taskEx != nil {
			scheduleOnActionComplete(ex, 0)
		}
		return
	}

	msg := fmt.Sprintf("Failed to complete action [error=%s, action=%s, task=%s]", err.Error(), exID, taskEx.ID)
	log.Error(taskEx.GetContext(), msg)

	act.Fail(msg)
	if taskEx != nil {
		forceFailTask(taskEx, msg, nil)
	}
}

func OnActionComplete(actEx *objects.ActionExecution, result *objects.ActionResult) {
	act := buildActionFromActionExecution(actEx)
	onCommonActionComplete(act, result, actEx.ID, actEx)
}

func OnWorkflowActionComplete(wfEx *objects.WorkflowExecution, result *objects.ActionResult) {
	act := buildActionFromWorkflowExecution(wfEx)
	onCommonActionComplete(act, result, "workflow:"+wfEx.ID, wfEx)
}
