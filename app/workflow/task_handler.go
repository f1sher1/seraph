package workflow

import (
	"fmt"
	"seraph/app/objects"
	"seraph/app/workflow/interfaces"
	"seraph/app/workflow/states"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"time"
)

func CheckAffectedTasks(task interfaces.Task) {
	taskEx := task.GetTaskExecution()
	ctx := taskEx.GetContext()
	taskEx, err := objects.QueryTaskExecutionByID(ctx, taskEx.ID)
	if err != nil {
		log.Errorf(ctx, "Failed to query task execution %s, error: %s", taskEx.WorkflowExecutionID, taskEx.ID)
		return
	}

	wfEx, err := taskEx.GetWorkflowExecution()
	if err != nil {
		log.Errorf(ctx, "Failed to query workflow execution %s of task execution %s, error: %s", taskEx.WorkflowExecutionID, taskEx.ID, err.Error())
		return
	}

	if states.IsCompleted(wfEx.State) {
		return
	}

	wfSpec, err := wfEx.GetSpec()
	if err != nil {
		log.Errorf(ctx, "Failed to query workflow spec on execution %s, error: %s", wfEx.ID, err.Error())
		return
	}
	flowCtrl, err := GetFlowController(wfEx, wfSpec)
	if err != nil {
		return
	}
	flowCtrl.SetWorkflowExecution(wfEx)

	affectedTaskExecs := flowCtrl.FindIndirectlyAffectedTaskExecutions(taskEx.Name)
	for _, taskEx := range affectedTaskExecs {
		go scheduleRefreshTaskState(ctx.Clone(), taskEx.ID, 0)
	}
}

func scheduleRefreshTaskState(ctx *contextx.Context, taskExID string, delay time.Duration) {
	time.Sleep(100 * time.Millisecond)
	objects.Transaction(ctx, func(subCtx *contextx.Context) error {
		taskEx, err := objects.QueryTaskExecutionByID(subCtx, taskExID)
		if err != nil {
			return err
		}

		if states.IsCompleted(taskEx.State) || states.IsRunning(taskEx.State) {
			return nil
		}

		wfEx, err := objects.QueryWorkflowExecutionByID(subCtx, taskEx.WorkflowExecutionID)
		if err != nil {
			return err
		}

		if states.IsCompleted(wfEx.State) {
			return nil
		}

		wfSpec, err := wfEx.GetSpec()
		if err != nil {
			return err
		}

		flowCtrl, err := GetFlowController(wfEx, wfSpec)
		if err != nil {
			return err
		}
		flowCtrl.SetWorkflowExecution(wfEx)

		return objects.WithNamedLock(subCtx, taskExID, func() error {
			taskEx, err := objects.QueryTaskExecutionByID(subCtx, taskExID)
			if err != nil {
				return err
			}

			if states.IsCompleted(taskEx.State) || states.IsRunning(taskEx.State) {
				return nil
			}

			logicSt := flowCtrl.GetLogicTaskState(taskEx)
			taskEx.RuntimeContext["triggered_by"] = logicSt.TriggeredBy

			if states.IsRunning(logicSt.State) {
				continueTask(taskEx)
			} else if states.IsErrored(logicSt.State) {
				completeTask(taskEx, logicSt.State, logicSt.StateInfo)
			} else if states.IsWaiting(logicSt.State) {
				log.Infof(subCtx, "Task execution is still in WAITING state [task_ex_id=%s, task_name=%s]", taskExID, taskEx.Name)
			} else {
				// Must never get here.
				log.Panicf(subCtx, "Unexpected logical task state [task_ex_id=%s, task_name=%s, state=%s]", taskExID, taskEx.Name, logicSt.State)
			}
			return nil
		})
	})

}

func buildTaskFromExecution(taskEx *objects.TaskExecution, wfEx *objects.WorkflowExecution) interfaces.Task {
	taskSpec, err := taskEx.GetSpec()
	if err != nil {
		return nil
	}

	var triggeredBySlice []objects.Table
	if triggeredBy, ok := taskEx.RuntimeContext["triggered_by"]; ok {
		// triggeredBySlice = triggeredBy.([]map[string]interface{})
		for _, v := range triggeredBy.([]interface{}) {
			triggeredBySlice = append(triggeredBySlice, v.(map[string]interface{}))
		}
	}

	task, err := NewTask(taskEx.GetContext(), wfEx, taskEx, taskSpec, taskEx.InContext, triggeredBySlice)
	if err != nil {
		return nil
	}

	return task
}

func completeTask(taskEx *objects.TaskExecution, state string, stateInfo string) {
	if taskEx == nil {
		return
	}

	wfEx, err := taskEx.GetWorkflowExecution()
	if err != nil {
		return
	}

	task := buildTaskFromExecution(taskEx, wfEx)
	if task == nil {
		return
	}

	if err := task.Complete(state, stateInfo); err != nil {
		task.ForceFail(fmt.Errorf("failed to complete task [error=%s, wf=%s, task=%s]", err.Error(), wfEx.ID, taskEx.ID))
		return
	}

	CheckAffectedTasks(task)
}

func continueTask(taskEx *objects.TaskExecution) {
	if taskEx == nil {
		return
	}

	wfEx, err := taskEx.GetWorkflowExecution()
	if err != nil {
		return
	}

	task := buildTaskFromExecution(taskEx, wfEx)
	if task == nil {
		return
	}

	if err := task.SetState(states.RUNNING, "", nil); err != nil {
		task.ForceFail(fmt.Errorf("failed to change task state from %s to %s [error=%s, wf=%s, task=%s]", taskEx.State, states.RUNNING, err.Error(), wfEx.ID, taskEx.ID))
		return
	}

	if err := task.Run(); err != nil {
		task.ForceFail(fmt.Errorf("failed to complete task [error=%s, wf=%s, task=%s]", err.Error(), wfEx.ID, taskEx.ID))
		return
	}

	CheckAffectedTasks(task)
}

func scheduleOnActionComplete(actOrWfEx interface{}, delay time.Duration) {
	var id string
	var ctx *contextx.Context
	var actEx *objects.ActionExecution
	var wfEx *objects.WorkflowExecution
	var taskEx *objects.TaskExecution
	var err error
	switch t := actOrWfEx.(type) {
	case *objects.ActionExecution:
		actEx = t
		id = actEx.ID
		ctx = actEx.GetContext()
		taskEx, err = actEx.GetTaskExecution()
		if err != nil {
			log.Errorf(ctx, "fetch task execution on action execution %s failed, error: %s", id, err.Error())
			return
		}
	case *objects.WorkflowExecution:
		wfEx = t
		id = actEx.ID
		ctx = wfEx.GetContext()
		taskEx, err = wfEx.GetTaskExecution()
		if err != nil {
			log.Errorf(ctx, "fetch task execution on workflow execution %s failed, error: %s", id, err.Error())
			return
		}
	default:
		log.Errorf(nil, "fetch task execution on execution %#v failed, error: invalid type", actOrWfEx)
		return
	}

	var taskSpec *objects.TaskSpec
	taskSpec, err = taskEx.GetSpec()
	if err != nil {
		log.Errorf(taskEx.GetContext(), "parse spec on task execution %s failed, error: %s", taskEx.ID, err.Error())
		return
	}

	log.Debugf(taskEx.GetContext(), "! %s", taskSpec.Name)
	if len(taskSpec.GetWithItems()) == 0 {
		if actEx != nil {
			onTaskActionComplete(actEx)
		} else {
			onTaskWorkflowActionComplete(wfEx)
		}
		return
	}

	go _scheduleOnActionComplete(ctx.Clone(), id, wfEx != nil)
}

func forceFailTask(taskEx *objects.TaskExecution, errMsg string, task interfaces.Task) {
	log.Errorf(taskEx.GetContext(), errMsg)

	wfEx, err := taskEx.GetWorkflowExecution()
	if err != nil {
		return
	}

	if task == nil {
		task = buildTaskFromExecution(taskEx, wfEx)
	}

	prevState := taskEx.State
	task.SetState(states.ERROR, errMsg, nil)
	task.Notify(prevState, states.ERROR)
	ForceFailWorkflow(wfEx, errMsg)
}

func onTaskActionComplete(actEx *objects.ActionExecution) {
	taskEx, err := actEx.GetTaskExecution()
	if err != nil {
		return
	}
	if taskEx == nil {
		return
	}

	wfEx, err := taskEx.GetWorkflowExecution()
	if err != nil {
		return
	}

	task := buildTaskFromExecution(taskEx, wfEx)
	err = task.OnActionComplete(actEx)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to handle action completion [error=%s, wf=%s, task=%s, action=%s]", err.Error(), wfEx.ID, taskEx.ID, actEx.ID)
		forceFailTask(taskEx, errMsg, task)
	} else {
		CheckAffectedTasks(task)
	}
}

func onTaskWorkflowActionComplete(wfEx *objects.WorkflowExecution) {
	taskEx, err := wfEx.GetTaskExecution()
	if err != nil {
		return
	}
	if taskEx == nil {
		return
	}

	_wfEx, err := taskEx.GetWorkflowExecution()
	if err != nil {
		return
	}

	task := buildTaskFromExecution(taskEx, _wfEx)
	err = task.OnActionComplete(wfEx)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to handle action completion [error=%s, wf=%s, task=%s, action=%s]", err.Error(), _wfEx.ID, taskEx.ID, wfEx.ID)
		forceFailTask(taskEx, errMsg, task)
	} else {
		CheckAffectedTasks(task)
	}
}

func _scheduleOnActionComplete(ctx *contextx.Context, exID string, isWfExecution bool) {
	time.Sleep(100 * time.Millisecond)
	err := objects.Transaction(ctx, func(subCtx *contextx.Context) error {
		if isWfExecution {
			wfEx, err := objects.QueryWorkflowExecutionByID(subCtx, exID)
			if err != nil {
				return err
			}
			onTaskWorkflowActionComplete(wfEx)
		} else {
			actEx, err := objects.QueryActionExecutionByID(subCtx, exID)
			if err != nil {
				return err
			}
			onTaskActionComplete(actEx)
		}
		return nil
	})
	if err != nil {
		log.Errorf(ctx, "schedule execution %s with error: %s", exID, err.Error())
	}
}
