package workflow

import (
	"seraph/app/expressions"
	"seraph/app/objects"
	"seraph/app/workflow/actions"
	"seraph/app/workflow/data_flow"
	"seraph/app/workflow/interfaces"
	"seraph/app/workflow/states"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"time"

	"github.com/google/uuid"
)

func NewTask(ctx *contextx.Context, wfExec *objects.WorkflowExecution, taskEx *objects.TaskExecution, taskSpec *objects.TaskSpec, taskCtx map[string]interface{}, triggeredBy []objects.Table) (interfaces.Task, error) {
	wfSpec, err := wfExec.GetSpec()
	if err != nil {
		return nil, err
	}

	if taskSpec == nil {
		taskSpec, err = taskEx.GetSpec()
		if err != nil {
			return nil, err
		}
	}

	withItems := taskSpec.GetWithItems()
	if len(withItems) > 0 {
		return &WithItemsTask{
			BaseTask: BaseTask{
				name:        taskSpec.Name,
				wfExec:      wfExec,
				wfSpec:      wfSpec,
				taskEx:      taskEx,
				taskSpec:    taskSpec,
				taskCtx:     taskCtx,
				ctx:         ctx,
				triggeredBy: triggeredBy,
			},
			withItems: withItems,
		}, nil
	}

	return &RegularTask{
		BaseTask: BaseTask{
			name:        taskSpec.Name,
			wfExec:      wfExec,
			wfSpec:      wfSpec,
			taskEx:      taskEx,
			taskSpec:    taskSpec,
			taskCtx:     taskCtx,
			ctx:         ctx,
			triggeredBy: triggeredBy,
		},
	}, nil
}

type BaseTask struct {
	name     string
	wfExec   *objects.WorkflowExecution
	wfSpec   *objects.WorkflowSpec
	taskEx   *objects.TaskExecution
	taskSpec *objects.TaskSpec
	taskCtx  map[string]interface{}

	uniqueKey   string
	waiting     bool
	stateChange bool
	triggeredBy []objects.Table

	ctx *contextx.Context
}

func (r *BaseTask) SetUniqueKey(s string) {
	r.uniqueKey = s
}

func (r *BaseTask) SetWait(b bool) {
	r.waiting = b
}

func (r BaseTask) ToMap() map[string]interface{} {
	data := map[string]interface{}{
		"name":                  r.name,
		"workflow_execution_id": r.wfExec.ID,
		"task_execution_id":     r.taskEx.ID,
		"context":               r.taskCtx,
		"unique_key":            r.uniqueKey,
		"triggered_by":          r.triggeredBy,
	}
	return data
}

func (r *BaseTask) createTaskExecution(state string, stateInfo string) (*objects.TaskExecution, error) {
	taskEx := objects.NewTaskExecution()
	taskEx.ID = uuid.NewString()
	taskEx.Name = r.taskSpec.Name
	taskEx.Type = r.taskSpec.GetType()
	taskEx.WorkflowExecutionID = r.wfExec.ID
	taskEx.WorkflowNamespace = r.wfExec.WorkflowNamespace
	taskEx.WorkflowID = r.wfExec.WorkflowID
	taskEx.State = state
	taskEx.StateInfo = stateInfo
	taskEx.Spec = r.taskSpec.ToString()
	taskEx.UniqueKey = r.uniqueKey
	taskEx.InContext = r.taskCtx
	taskEx.Published = map[string]interface{}{}
	taskEx.RuntimeContext = map[string]interface{}{}
	taskEx.ProjectID = r.wfExec.ProjectID
	if r.triggeredBy != nil {
		taskEx.RuntimeContext["triggered_by"] = r.triggeredBy
	}
	err := taskEx.Save(r.ctx)
	if err != nil {
		log.Errorf(r.ctx, "create task execution failed, error: %s", err.Error())
		return nil, err
	}
	return taskEx, nil
}

func (r *BaseTask) deferTask() {
	if r.taskEx == nil {
		taskExs, err := objects.QueryTaskExecutions(r.ctx, r.wfExec.ID, r.uniqueKey, states.WAITING, nil, nil)
		if err != nil {
			log.Errorf(r.ctx, "can't get waiting task executions which belongs to workflow execution %s", r.wfExec.ID)
			return
		}

		if len(taskExs) > 0 {
			r.taskEx = taskExs[0]
		}
	}

	if r.taskEx != nil {
		return
	}

	err := objects.WithNamedLock(r.ctx, r.uniqueKey, func() error {
		if r.taskEx == nil {
			taskExs, err := objects.QueryTaskExecutions(r.ctx, r.wfExec.ID, r.uniqueKey, states.WAITING, nil, nil)

			if err != nil {
				log.Errorf(r.ctx, "can't get waiting task executions which belongs to workflow execution %s", r.wfExec.ID)
				return err
			}

			if len(taskExs) > 0 {
				r.taskEx = taskExs[0]
			}
		}

		stateInfo := "Task is waiting"
		if r.taskEx == nil {
			taskEx, err := r.createTaskExecution(states.WAITING, stateInfo)
			if err != nil {
				return err
			}
			r.taskEx = taskEx
		} else if r.taskEx.State != states.WAITING {
			return r.SetState(states.WAITING, stateInfo, nil)
		}

		return nil
	})

	if err != nil {
		log.Errorf(r.ctx, "deferTask error: %s", err.Error())
	}

}

func (r *BaseTask) SetState(state string, stateInfo string, processed interface{}) error {
	curState := r.taskEx.State

	if curState == states.WAITING && state == states.RUNNING {
		r.taskEx.StartedAt = time.Now()
	}

	if curState != state || r.taskEx.StateInfo != stateInfo {
		taskEx, err := objects.QueryTaskExecutionByID(r.ctx, r.taskEx.ID)
		if err != nil {
			log.Errorf(r.ctx, "save task execution %s state from %s to %s failed, error: %s", r.taskEx.ID, curState, state, err.Error())
			return err
		}

		if taskEx.State != curState {
			// 状态不一致，跳过设置
			return nil
		}

		taskEx.State = state
		taskEx.StateInfo = stateInfo
		r.stateChange = true

		if states.IsCompleted(state) {
			taskEx.FinishedAt = time.Now()
		}

		if processed != nil {
			taskEx.Processed = processed.(bool)
		}
		err = taskEx.Update(r.ctx, "State", "StateInfo", "FinishedAt", "Processed")
		if err != nil {
			log.Errorf(r.ctx, "save task execution %s state from %s to %s failed, error: %s", r.taskEx.ID, curState, state, err.Error())
			return err
		}

		r.taskEx = taskEx
		log.Infof(r.ctx, "Task '%s' (%s) [%s -> %s, msg=%s]", r.taskEx.Name, r.taskEx.ID, curState, state, r.taskEx.StateInfo)
		return nil
	}

	return nil
}

func (r *BaseTask) evaluateExpression(inputSpec interface{}, ctx objects.Table) (interface{}, error) {
	// NOTE:(sun) 增加上个节点结果是否作为入参
	var parentResult map[string]interface{}
	if r.taskSpec.GetParentResult.IsGetParentResult {
		parentResult = map[string]interface{}{
			"parent_result": r.taskSpec.GetParentResult.Result,
		}
	}
	dataCtx := data_flow.NewDataContext(
		data_flow.GetTaskInfo(r.taskEx),
		data_flow.GetWorkflowEnvironment(r.wfExec),
		ctx,
		r.wfExec.Context,
		r.wfExec.Input,
		r.taskEx.InContext,
		parentResult,
	)
	return expressions.EvaluateRecursively(inputSpec, dataCtx)
}

func (r *BaseTask) getActionDefaults() objects.Table {
	data := objects.Table{}
	actionName := r.taskSpec.Name
	if actionName != "" {
		if env, ok := r.wfExec.Params["__actions"]; ok {
			if actionEnv, ok := env.(map[string]interface{})[actionName]; ok {
				for k, v := range actionEnv.(map[string]interface{}) {
					data[k] = v
				}
			}
		}
	}

	return data
}

func (r *BaseTask) getActionInput() (objects.Table, error) {
	input, err := r.evaluateExpression(r.taskSpec.GetParameters(), r.taskCtx)
	if err != nil {
		return nil, err
	}

	data := objects.Table{}
	for k, v := range r.getActionDefaults() {
		data[k] = v
	}
	for k, v := range input.(map[string]interface{}) {
		data[k] = v
	}

	return data, nil
}

func (r *BaseTask) getAction() (interfaces.TaskAction, error) {
	wfName := r.taskSpec.GetWorkflowName()
	if wfName != "" {
		return actions.NewWorkflowAction(wfName, r.taskEx, nil)
	} else {
		actionName := r.taskSpec.GetActionName()
		if actionName != "" {
			// For dynamic action evaluation we just regenerate the name.
			_actionName, err := r.evaluateExpression(actionName, nil)
			if err != nil {
				return nil, err
			}
			actionName = _actionName.(string)
		}

		if actionName == "" {
			actionName = "std.noop"
		}

		if actionDef, err := actions.ResolveActionDefinition(r.ctx, actionName, r.wfExec.Name, r.wfSpec.Name); err != nil {
			return nil, err
		} else {
			var action interfaces.TaskAction
			if actionDef.Spec != "" {
				// 嵌套
				action, err = actions.NewAdHocAction(r.ctx, actionDef, r.taskEx, r.taskCtx, objects.Table(r.wfExec.Context))
			} else {
				// 可以直接执行
				action, err = actions.NewGolangAction(r.ctx, actionDef, nil, r.taskEx)
			}
			if err != nil {
				return nil, err
			}
			return action, err
		}
	}
}

func (r *BaseTask) Notify(oldState string, newState string) {
	// instance_uuid := r.wfExec.Name
	// now := time.Now()
	// msg := unimq.Msg{
	// 	ContextRequestId:     r.wfExec.ID,
	// 	ContextRequestAction: r.taskEx.Name,
	// 	ContextRoles:         []string{"admin"},
	// 	ContextIsAdmin:       true,
	// 	ContextReadDeleted:   "no",
	// 	ContextProjectId:     r.ctx.GetProjectID(),
	// 	EventType:            fmt.Sprintf("seraph.%v.%v", r.taskEx.Name, strings.ToLower(newState)),
	// 	Priority:             "INFO",
	// 	Payload: map[string]interface{}{
	// 		"instance_id": instance_uuid,
	// 		"status":      newState,
	// 		"task_name":   r.taskEx.Name,
	// 		"task_id":     r.taskEx.ID,
	// 	},
	// 	Timestamp: now.Format("2006-01-01T15:04:05.000"),
	// 	UniqueId:  fmt.Sprintf("wf-%v", uuid.NewString()),
	// }
	log.Debug(r.ctx, "******")
	// unimq.DeliverMessage(msg)
}

func (r *BaseTask) GetTaskExecution() *objects.TaskExecution {
	panic("implement me")
}

func (r *BaseTask) Complete(state string, stateInfo string) error {
	oldState := r.taskEx.State
	if states.IsCompleted(oldState) {
		r.Notify(oldState, oldState)
		return nil
	}

	err := r.SetState(state, stateInfo, nil)
	if err != nil {
		return err
	}

	data_flow.PublishVariables(r.taskEx, r.taskSpec)

	// Ignore DELAYED state.
	if states.IsDelayed(r.taskEx.State) {
		return nil
	}

	flowCtrl, err := GetFlowController(r.wfExec, r.wfSpec)
	if err != nil {
		return err
	}
	flowCtrl.SetWorkflowExecution(r.wfExec)

	nextTasks, err := flowCtrl.GetNextTasksForTask(r.taskEx)
	if err != nil {
		return err
	}
	r.taskEx.NextTasks = []interface{}{}

	for _, t := range nextTasks {
		event := t.GetTriggerEvent()
		tSpec := t.GetTaskSpec()
		r.taskEx.NextTasks = append(r.taskEx.NextTasks, []string{
			tSpec.Name, event,
		})
		// NOTE:(sun)获取上个task的result
		if tSpec.GetParentResult.IsGetParentResult {
			tSpec.GetParentResult.Result = stateInfo
		}

		// save when error was handled
		if t.HandledErrors() {
			r.taskEx.ErrorHandled = true
		}
	}
	r.taskEx.HasNextTasks = len(r.taskEx.NextTasks) > 0

	if states.IsPaused(r.wfExec.State) {
		r.Notify(oldState, r.taskEx.State)
		return nil
	}

	r.taskEx.Processed = true
	r.taskEx.FinishedAt = time.Now().UTC()
	// NOTE:二次更新
	// HACK: 时间类型的字段无法更新
	r.taskEx.Update(r.ctx, "Processed", "FinishedAt", "NextTasks", "HasNextTasks", "ErrorHandled")
	r.Notify(oldState, r.taskEx.State)

	if flowCtrl.MayCompleteWorkflow(r.taskEx) {
		go func(tx *contextx.Context) {
			time.Sleep(100 * time.Millisecond)
			CheckAndCompleteWorkflow(tx.Clone(), r.wfExec.ID)
		}(r.ctx)
	}

	DispatchWorkflowTasks(r.wfExec, nextTasks)
	return nil
}

func (r *BaseTask) OnActionComplete(ex interface{}) error {
	panic("implement me")
}

func (r *BaseTask) Run() error {
	panic("implement me")
}

func (r *BaseTask) ForceFail(err error) {
	panic("implement me")
}

func (r *BaseTask) GetTriggerEvent() string {
	if len(r.triggeredBy) > 0 {
		if eventName, ok := r.triggeredBy[0]["event"]; ok {
			return eventName.(string)
		}
	}
	//if triggeredBy, ok := r.taskEx.RuntimeContext["triggered_by"]; ok {
	//	triggeredBySlice := triggeredBy.([]map[string]interface{})
	//	if len(triggeredBySlice) > 0 {
	//		if eventName, ok := triggeredBySlice[0]["event"]; ok {
	//			return eventName.(string)
	//		}
	//	}
	//}
	return ""
}

func (r *BaseTask) GetTaskSpec() *objects.TaskSpec {
	return r.taskSpec
}

func (r *BaseTask) HandledErrors() bool {
	return true
}
