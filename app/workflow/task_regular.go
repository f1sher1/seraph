package workflow

import (
	"seraph/app/objects"
	"seraph/app/workflow/states"
	"seraph/pkg/log"
	"time"
)

type RegularTask struct {
	BaseTask
}

func (r *RegularTask) GetTaskExecution() *objects.TaskExecution {
	return r.taskEx
}

func (r *RegularTask) OnActionComplete(ex interface{}) error {
	var state, stateInfo string
	switch ex.(type) {
	case *objects.WorkflowExecution:
		wfEx := ex.(*objects.WorkflowExecution)
		state = wfEx.State
		if !states.IsSuccess(state) {
			if actionResult, ok := wfEx.Output["result"]; ok {
				stateInfo = actionResult.(string)
			}
		}
	case *objects.ActionExecution:
		actEx := ex.(*objects.ActionExecution)
		state = actEx.State
		if !states.IsSuccess(state) {
			if actionResult, ok := actEx.Outputs["result"]; ok {
				stateInfo = actionResult.(string)
			}
		}
	}
	return r.Complete(state, stateInfo)
}

func (r *RegularTask) Run() error {
	if r.taskEx == nil {
		return r.RunNew()
	}
	return r.RunExisting()
}

func (r *RegularTask) RunNew() error {
	if r.waiting {
		defer r.deferTask()
		return nil
	}

	taskEx, err := r.createTaskExecution(states.IDLE, "")
	if err != nil {
		return err
	}
	r.taskEx = taskEx

	action, err := r.getAction()
	if err != nil {
		return err
	}

	input, err := r.getActionInput()
	if err != nil {
		return err
	}

	if err := action.ValidateInput(input); err != nil {
		return err
	}

	return action.Schedule(input, r.taskSpec.Target, time.Duration(r.taskSpec.Timeout)*time.Millisecond, 0, "")
}

func (r *RegularTask) RunExisting() error {
	if r.waiting {
		return nil
	}

	action, err := r.getAction()
	if err != nil {
		return err
	}

	input, err := r.getActionInput()
	if err != nil {
		return err
	}

	if err := action.ValidateInput(input); err != nil {
		return err
	}

	return action.Schedule(input, r.taskSpec.Target, time.Duration(r.taskSpec.Timeout)*time.Millisecond, 0, "")
}

func (r *RegularTask) ForceFail(err error) {
	oldState := r.taskEx.State
	setErr := r.SetState(states.ERROR, err.Error(), true)
	if setErr != nil {
		log.Errorf(r.ctx, "Set state failed, error: %s", setErr.Error())
	}
	r.Notify(oldState, states.ERROR)
}
