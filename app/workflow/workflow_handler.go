package workflow

import (
	"fmt"
	"seraph/app/objects"
	"seraph/app/workflow/states"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
)

func StopWorkflow(wfEx *objects.WorkflowExecution, state, errMsg string) {
	wf := Workflow{
		Execution: wfEx,
		Ctx:       wfEx.GetContext(),
	}
	wf.Initialize()
	wf.Stop(state, errMsg)

	if states.IsCanceled(state) {
		taskExs, err := wfEx.GetChildrenTaskExecutions()
		if err != nil {
			panic(err)
		}
		for _, ex := range taskExs {
			subWfEx, err := objects.QueryWorkflowExecutionByTaskExecutionID(wfEx.GetContext(), ex.ID)
			if err != nil {
				panic(err)
			}
			if subWfEx != nil && !states.IsCompleted(subWfEx.State) {
				StopWorkflow(subWfEx, state, errMsg)
			}
		}
	}
}

func ForceFailWorkflow(wfEx *objects.WorkflowExecution, err string) {
	StopWorkflow(wfEx, states.ERROR, err)
}

func CheckAndCompleteWorkflow(ctx *contextx.Context, wfExID string) {
	wfEx, err := objects.QueryWorkflowExecutionByID(ctx, wfExID)
	if wfEx == nil || states.IsCompleted(wfEx.State) {
		return
	}

	if err != nil {
		log.Errorf(ctx, "error happened when check and complete workflow %s, error: %s", wfExID, err.Error())
		return
	}

	wf := Workflow{
		Ctx:       ctx,
		Execution: wfEx,
	}
	err = wf.Initialize()
	if err != nil {
		log.Errorf(ctx, "error happened when check and complete workflow %s, error: %s", wfExID, err.Error())
		return
	}

	err = wf.CheckAndComplete()
	if err != nil {
		ForceFailWorkflow(wfEx, fmt.Sprintf("Failed to check and complete [wf_ex_id=%s, wf_name=%s]:", wfExID, wfEx.Name))
	}
}
