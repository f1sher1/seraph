package workflow

import (
	"encoding/json"
	"fmt"
	"reflect"
	"seraph/app/engine/client"
	"seraph/app/expressions"
	"seraph/app/objects"
	"seraph/app/unimq"
	"seraph/app/workflow/data_flow"
	"seraph/app/workflow/interfaces"
	"seraph/app/workflow/states"
	"seraph/pkg/contextx"
	"seraph/pkg/gormx"
	"seraph/pkg/log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Workflow struct {
	Execution *objects.WorkflowExecution
	Spec      *objects.WorkflowSpec
	Ctx       *contextx.Context

	flowController interfaces.WorkflowController
}

func (w *Workflow) InitializeSpec() error {
	if w.Execution != nil {
		spec, err := w.Execution.GetSpec()
		if err != nil {
			return err
		}
		w.Spec = spec
	}
	return nil
}

func (w *Workflow) InitializeController() error {
	ctrl, err := InitializeFlowController(w.Ctx, w.Spec)
	if err != nil {
		return err
	}
	w.flowController = ctrl
	if w.Execution != nil {
		w.flowController.SetWorkflowExecution(w.Execution)
	}
	return nil
}

func (w *Workflow) Initialize() error {
	if err := w.InitializeSpec(); err != nil {
		return err
	}
	if err := w.InitializeController(); err != nil {
		return err
	}
	return nil
}

func (w *Workflow) createDBEntry(id, description, scope string, tags []string, input objects.Table, params objects.Table) error {
	if w.Execution != nil {
		return nil
	}
	w.Execution = objects.NewWorkflowExecution()
	w.Execution.ID = id
	w.Execution.Name = w.Spec.Name
	w.Execution.Description = description
	w.Execution.ProjectID = w.Ctx.GetProjectID()
	w.Execution.State = states.IDLE
	if scope != "" {
		w.Execution.Scope = scope
	}
	if tags != nil {
		for _, tag := range tags {
			w.Execution.Tags = append(w.Execution.Tags, tag)
		}
	}
	if input != nil {
		w.Execution.Input = map[string]interface{}{}
		for k, v := range input {
			w.Execution.Input[k] = v
		}
	}
	if params != nil {
		w.Execution.Params = map[string]interface{}{}
		for k, v := range params {
			w.Execution.Params[k] = v
		}
	}
	specBytes, err := json.Marshal(w.Spec)
	if err != nil {
		return err
	}
	w.Execution.Spec = string(specBytes)

	return w.Execution.Save(w.Ctx)
}

func (w *Workflow) ValidateInput(input objects.Table) error {
	w.Spec.GetInputNames()
	return nil
}

func (w *Workflow) Start(id, description, scope string, tags []string, input objects.Table, params objects.Table) error {
	log.Debugf(w.Ctx, "Starting workflow [name=%s, input=%s]", w.Spec.Name, input.ToString())

	// 验证输入参数是否正确
	if err := w.ValidateInput(input); err != nil {
		return err
	}

	// 创建数据库记录
	if w.Execution == nil {
		if err := w.createDBEntry(id, description, scope, tags, input, params); err != nil {
			return err
		}
	}
	w.flowController.SetWorkflowExecution(w.Execution)

	// 设置执行状态
	if err := w.SetState(states.RUNNING, ""); err != nil {
		return err
	}

	nextTasks, err := w.flowController.GetNextTasks(nil)
	if err != nil {
		return err
	}

	return DispatchWorkflowTasks(w.Execution, nextTasks)
}

func (w *Workflow) SetState(state string, stateInfo string) error {
	curState := w.Execution.State

	if err := states.ValidateStateTransition(curState, state); err != nil {
		return err
	}

	//taskEx, err := objects.QueryTaskExecutionByID(w.ctx, w.Execution.ID)
	//if err != nil {
	//	return err
	//}
	//
	//taskEx.State = state
	//err = taskEx.Save(w.ctx)
	//if err != nil {
	//	return err
	//}

	var changeFields []string
	w.Execution.State = state
	w.Execution.StateInfo = stateInfo
	changeFields = append(changeFields, "State", "StateInfo")
	if states.IsCompleted(state) {
		w.Execution.FinishedAt = time.Now().UTC()
		changeFields = append(changeFields, "FinishedAt")
	}
	// err := w.Execution.Save(w.Ctx)
	err := w.Execution.Update(w.Ctx, changeFields...)

	log.Debugf(w.Ctx, "workflow '%s' [%s -> %s, msg=%s]", w.Execution.WorkflowName, curState, state, stateInfo)
	return err
}

func (w *Workflow) Pause() error {
	if err := w.SetState(states.PAUSED, ""); err != nil {
		return err
	}
	return nil
}

func (w *Workflow) Resume() error {
	if err := w.SetState(states.RUNNING, ""); err != nil {
		return err
	}
	return nil
}

func (w *Workflow) Stop(state string, err string) error {
	if states.IsSuccess(state) {
		return w.SucceedWorkflow(w.GetFinalContext(), err)
	} else if states.IsErrored(state) {
		return w.FailWorkflow(w.GetFinalContext(), err)
	} else if states.IsCanceled(state) {
		return w.CancelWorkflow(err)
	} else {
		return fmt.Errorf("invalid state %s", state)
	}
}

func (w *Workflow) GetFinalContext() map[string]interface{} {
	var finalCtx map[string]interface{}

	evalCtx, err := w.flowController.EvaluateWorkflowFinalContext()
	if err != nil {
		log.Warnf(w.Ctx, "Failed to get final context for workflow execution. [wf_ex_id: %s, wf_name: %s, error: %s]", w.Execution.ID, w.Execution.Name, err.Error())
	} else {
		for k, v := range evalCtx {
			finalCtx[k] = v
		}
	}
	return finalCtx
}

func (w *Workflow) SucceedWorkflow(finalCtx map[string]interface{}, msg string) error {
	dataCtx := data_flow.NewDataContext(
		w.Ctx.GetMap(),
		data_flow.GetWorkflowEnvironment(w.Execution),
		w.Execution.Context,
		w.Execution.Input,
	)
	output, err := expressions.EvaluateRecursively(finalCtx, dataCtx)
	if err != nil {
		return err
	}

	switch reflect.ValueOf(output).Kind() {
	case reflect.Slice:
		if len(output.([]interface{})) == 0 {
			output = w.Ctx.GetMap()
		}
	case reflect.Map:
		if convOutput, ok := output.(objects.Table); ok {
			if len(convOutput) == 0 {
				output = w.Ctx.GetMap()
			}
		}
	}

	if err := w.SetState(states.SUCCESS, msg); err != nil {
		return nil
	}

	w.Execution.Output = output.(map[string]interface{})
	// todo; notify success event
	now := time.Now()
	m := unimq.Msg{
		ContextRequestId:     w.Execution.ID,
		ContextRequestAction: w.Execution.Name,
		ContextRoles:         []string{"admin"},
		ContextIsAdmin:       true,
		ContextReadDeleted:   "no",
		ContextProjectId:     w.Ctx.GetProjectID(),
		EventType:            fmt.Sprintf("seraph.wf-%v.success", w.Execution.ID),
		Priority:             "INFO",
		Payload: map[string]interface{}{
			"instance_id": w.Execution.Name,
			"status":      "success",
			"workflow_id": w.Execution.ID,
		},
		Timestamp: now.Format("2006-01-01T15:04:05.000"),
		UniqueId:  fmt.Sprintf("wf-%v", uuid.NewString()),
	}
	log.Debugf(w.Ctx, "workflow notify: %#v", m)
	// unimq.DeliverMessage(m)

	if w.Execution.TaskExecutionID != "" {
		w.SendResultToParentWorkflow()
	}
	return nil
}

func (w *Workflow) FailWorkflow(finalCtx map[string]interface{}, msg string) error {
	dataCtx := data_flow.NewDataContext(
		w.Ctx.GetMap(),
		data_flow.GetWorkflowEnvironment(w.Execution),
		w.Execution.Context,
		w.Execution.Input,
	)
	output, err := expressions.EvaluateRecursively(finalCtx, dataCtx)
	if err != nil {
		msg = fmt.Sprintf("Failed to evaluate expression in output-on-error! (output-on-error: '%s', exception: '%s' Cause: '%s'", dataCtx, err.Error(), msg)
	}

	switch reflect.ValueOf(output).Kind() {
	case reflect.Slice:
		if len(output.([]interface{})) == 0 {
			output = w.Ctx.GetMap()
		}
	case reflect.Map:
		//HACK: interface conversion: interface {} is map[string]interface {}, not objects.Table
		if len(output.(map[string]interface{})) == 0 {
			output = w.Ctx.GetMap()
		}
	}

	if err := w.SetState(states.ERROR, msg); err != nil {
		return nil
	}

	errOutput := objects.Table{
		"result": msg,
	}
	output = errOutput.MergeToNew(output.(map[string]interface{}))

	// HACK: interface conversion: interface {} is objects.Table, not map[string]interface {}
	w.Execution.Output = (gormx.MapJson)(output.(objects.Table))
	// todo; notify failure event
	now := time.Now()
	m := unimq.Msg{
		ContextRequestId:     w.Execution.ID,
		ContextRequestAction: w.Execution.Name,
		ContextRoles:         []string{"admin"},
		ContextIsAdmin:       true,
		ContextReadDeleted:   "no",
		ContextProjectId:     w.Ctx.GetProjectID(),
		EventType:            fmt.Sprintf("seraph.wf-%v.failed", w.Execution.ID),
		Priority:             "INFO",
		Payload: map[string]interface{}{
			"instance_id": w.Execution.Name,
			"status":      "failed",
			"workflow_id": w.Execution.ID,
		},
		Timestamp: now.Format("2006-01-01T15:04:05.000"),
		UniqueId:  fmt.Sprintf("wf-%v", uuid.NewString()),
	}
	log.Debugf(w.Ctx, "workflow notify: %#v", m)
	// unimq.DeliverMessage(m)

	if w.Execution.TaskExecutionID != "" {
		w.SendResultToParentWorkflow()
	}
	return nil
}

func (w *Workflow) CancelWorkflow(msg string) error {
	if states.IsCompleted(w.Execution.State) {
		return nil
	}
	if err := w.SetState(states.CANCELLED, msg); err != nil {
		return nil
	}
	w.Execution.Output["result"] = msg
	// todo; notify cancel event
	now := time.Now()
	m := unimq.Msg{
		ContextRequestId:     w.Execution.ID,
		ContextRequestAction: w.Execution.Name,
		ContextRoles:         []string{"admin"},
		ContextIsAdmin:       true,
		ContextReadDeleted:   "no",
		ContextProjectId:     w.Ctx.GetProjectID(),
		EventType:            fmt.Sprintf("seraph.wf-%v.cancel", w.Execution.ID),
		Priority:             "INFO",
		Payload: map[string]interface{}{
			"instance_id": w.Execution.Name,
			"status":      "cancel",
			"workflow_id": w.Execution.ID,
		},
		Timestamp: now.Format("2006-01-01T15:04:05.000"),
		UniqueId:  fmt.Sprintf("wf-%v", uuid.NewString()),
	}
	log.Debugf(w.Ctx, "workflow notify: %#v", m)
	// unimq.DeliverMessage(m)

	if w.Execution.TaskExecutionID != "" {
		w.SendResultToParentWorkflow()
	}
	return nil
}

func (w *Workflow) SendResultToParentWorkflow() {
	var result *objects.ActionResult
	if states.IsSuccess(w.Execution.State) {
		result = nil
	} else if states.IsErrored(w.Execution.State) {
		msg := w.Execution.StateInfo
		if msg == "" {
			msg = fmt.Sprintf("Failed subworkflow [execution_id=%s]", w.Execution.ID)
		}
		result = &objects.ActionResult{Err: msg}
	} else if states.IsCanceled(w.Execution.State) {
		msg := w.Execution.StateInfo
		if msg == "" {
			msg = fmt.Sprintf("Cancelled subworkflow [execution_id=%s]", w.Execution.ID)
		}
		result = &objects.ActionResult{Err: msg, Cancel: true}
	} else {
		log.Warnf(w.Ctx, "Method SendResultToParentWorkflow() must never be called if a workflow is not in SUCCESS, ERROR or CANCELLED state [execution_id=%s].", w.Execution.ID)
		return
	}

	c, err := client.NewClient()
	if err != nil {
		log.Errorf(w.Ctx, "Fetch engine client failed [execution_id=%s], error: %s", w.Execution.ID, err.Error())
		return
	}
	_, err = c.OnActionComplete(w.Ctx, w.Execution.ID, result, true)
	if err != nil {
		log.Errorf(w.Ctx, "notify engine workflow action complete failed [execution_id=%s], error: %s", w.Execution.ID, err.Error())
		return
	}
}

func (w *Workflow) CheckAndComplete() error {
	if states.IsPaused(w.Execution.State) || states.IsCompleted(w.Execution.State) {
		return nil
	}

	taskExs, err := objects.QueryIncompleteTaskExecutions(w.Ctx, w.Execution.ID)
	if err != nil {
		return err
	}

	if len(taskExs) > 0 {
		log.Debugf(w.Ctx, "there are %d task executions[id=%s] is running, skip complete it", len(taskExs), w.Execution.ID)
		return nil
	}

	log.Debugf(w.Ctx, "Workflow completed [id=%s]", w.Execution.ID)

	if w.flowController.AnyCancels() {
		msg := _buildCancelInfoMessage(w.Ctx, w.flowController, w.Execution)
		return w.CancelWorkflow(msg)
	} else if w.flowController.AllErrorHandled() {
		finalCtx := w.GetFinalContext()
		return w.SucceedWorkflow(finalCtx, "")
	} else {
		msg := _buildFailInfoMessage(w.Ctx, w.flowController, w.Execution)
		finalCtx := w.GetFinalContext()
		return w.FailWorkflow(finalCtx, msg)
	}
}

func _buildCancelInfoMessage(ctx *contextx.Context, flowCtrl interfaces.WorkflowController, wfEx *objects.WorkflowExecution) string {
	canceledTasks, err := objects.QueryTaskExecutions(ctx, wfEx.ID, nil, states.CANCELLED, nil, nil)
	if err != nil {
		return err.Error()
	}

	var canceledTaskNames []string
	for _, t := range canceledTasks {
		canceledTaskNames = append(canceledTaskNames, t.Name)
	}

	return fmt.Sprintf("Cancelled tasks: %s", strings.Join(canceledTaskNames, ","))
}

func _buildFailInfoMessage(ctx *contextx.Context, flowCtrl interfaces.WorkflowController, wfEx *objects.WorkflowExecution) string {
	errTasks, err := objects.QueryTaskExecutions(ctx, wfEx.ID, nil, states.CANCELLED, nil, nil)
	if err != nil {
		return err.Error()
	}

	var errTaskNames []string
	for _, t := range errTasks {
		if !flowCtrl.IsErrorHandled(t) {
			errTaskNames = append(errTaskNames, t.Name)
		}
	}

	return fmt.Sprintf("Failure caused by error in tasks: %s", strings.Join(errTaskNames, ","))
}
