package workflow

import (
	"fmt"
	"seraph/app/expressions"
	"seraph/app/objects"
	"seraph/app/workflow/data_flow"
	"seraph/app/workflow/states"
	"seraph/pkg/log"
	"time"
)

var (
	withItemContextKey     = "with_items"
	withItemCountKey       = "count"
	withItemConcurrencyKey = "concurrency"
	withItemCapacityKey    = "capacity"
	defaultWithItemContext = map[string]int64{
		withItemCountKey:       0,
		withItemConcurrencyKey: 0,
		withItemCapacityKey:    0,
	}
)

type WithItemsTask struct {
	BaseTask
	withItems map[string]interface{}
}

func (t *WithItemsTask) ForceFail(err error) {
	oldState := t.taskEx.State
	setErr := t.SetState(states.ERROR, err.Error(), true)
	if setErr != nil {
		log.Errorf(t.ctx, "Set state failed, error: %s", setErr.Error())
	}
	t.Notify(oldState, states.ERROR)
}

func (t *WithItemsTask) Complete(state string, stateInfo string) error {
	panic("implement me")
}

func (t *WithItemsTask) OnActionComplete(ex interface{}) error {
	return objects.WithNamedLock(t.ctx, "with-items-"+t.taskEx.ID, func() error {
		taskEx, err := objects.QueryTaskExecutionByID(t.ctx, t.taskEx.ID)
		if err != nil {
			return err
		}
		t.taskEx = taskEx

		if states.IsCompleted(t.taskEx.State) {
			return nil
		}

		t.increaseCapacity()

		if t.isWithItemsCompleted() {
			var stateInfo string
			stateInfoMap := map[string]string{
				states.SUCCESS:   "",
				states.ERROR:     "One or more actions had failed.",
				states.CANCELLED: "One or more actions was cancelled.",
			}
			state, err := t.getFinalState()
			if err != nil {
				state = states.ERROR
				stateInfo = fmt.Sprintf("get final state failed, error: %s", err.Error())
			} else {
				stateInfo = stateInfoMap[state]
			}

			t.Complete(state, stateInfo)

		} else if t.hasMoreActionExecutions() && t.getWithItemConcurrency() > 0 {
			t.scheduleActions()
		}

		return nil
	})
}

func (t *WithItemsTask) hasMoreActionExecutions() bool {
	actionExs, err := t.taskEx.GetActionExecutions()
	if err != nil {
		return false
	}

	runningActionExCount := int64(0)
	for _, act := range actionExs {
		if states.IsRunning(act.State) {
			runningActionExCount += 1
		}
	}

	return t.getWithItemCount() > runningActionExCount
}

func (t *WithItemsTask) increaseCapacity() error {
	ctx := t.getWithItemContext()
	concurrency := t.getWithItemConcurrency()
	if concurrency > 0 && ctx[withItemCapacityKey] < concurrency {
		ctx[withItemCapacityKey] += 1
		t.taskEx.RuntimeContext[withItemContextKey] = ctx[withItemCapacityKey]
		t.taskEx.Save(nil)
	}
	return nil
}

func (t *WithItemsTask) getWithItemValues() ([]objects.Table, error) {
	var values []objects.Table

	result, err := t.evaluateExpression(t.taskSpec.GetWithItems(), nil)
	if err != nil {
		return nil, err
	}

	lastLen := 0
	_values := map[string][]interface{}{}
	for k, sliceV := range result.(objects.Table) {
		for _, v := range sliceV.([]interface{}) {
			if _, ok := _values[k]; ok {
				_values[k] = append(_values[k], v)
			} else {
				_values[k] = []interface{}{v}
			}
		}
		if lastLen > 0 {
			// 检查是不是长度一致
			if len(_values[k]) != lastLen {
				return nil, fmt.Errorf("with-item values length is different")
			}
		} else {
			lastLen = len(_values[k])
		}
	}

	if len(_values) == 0 {
		return values, nil
	}

	for i := 0; i < lastLen; i++ {
		for k, sliceV := range _values {
			if i == 0 {
				values = append(values, objects.Table{k: sliceV[i]})
			} else {
				values[i][k] = sliceV[i]
			}
		}
	}

	return values, nil
}

func (t WithItemsTask) getWithItemContext() map[string]int64 {
	ctx, ok := t.taskEx.RuntimeContext[withItemContextKey]
	if !ok {
		return defaultWithItemContext
	}
	return ctx.(map[string]int64)
}

func (t WithItemsTask) getWithItemCount() int64 {
	ctx := t.getWithItemContext()
	return ctx[withItemCountKey]
}

func (t WithItemsTask) getWithItemCapacity() int64 {
	ctx := t.getWithItemContext()
	return ctx[withItemCapacityKey]
}

func (t *WithItemsTask) getWithItemConcurrency() int64 {
	if concurrency, ok := t.taskEx.RuntimeContext[withItemConcurrencyKey]; ok {
		return concurrency.(int64)
	}
	return 0
}

func (t WithItemsTask) isWithItemsCompleted() bool {
	actionExs, err := t.taskEx.GetActionExecutions()
	if err != nil {
		return false
	}

	var finishedExs []*objects.ActionExecution
	// 查询是否有action execution是canceled
	for _, ex := range actionExs {
		if states.IsCanceled(ex.State) {
			return true
		} else if states.IsCompleted(ex.State) {
			finishedExs = append(finishedExs, ex)
		}
	}

	count := t.getWithItemCount()
	if count == 0 {
		count = 1
	}

	fullCapacity := false
	concurrency := t.getWithItemConcurrency()
	if concurrency == 0 || t.getWithItemCapacity() == concurrency {
		fullCapacity = true
	}
	return int64(len(finishedExs)) == count && fullCapacity
}

func (t WithItemsTask) getFinalState() (string, error) {
	actionExs, err := t.taskEx.GetActionExecutions()
	if err != nil {
		return "", err
	}

	// 检查是否有canceled
	for _, ex := range actionExs {
		if states.IsCanceled(ex.State) {
			return states.CANCELLED, nil
		}
	}

	// 检查是否有error
	for _, ex := range actionExs {
		if states.IsErrored(ex.State) {
			return states.ERROR, nil
		}
	}

	return states.SUCCESS, nil
}

func (t *WithItemsTask) prepareRuntimeContext(actionCount int64) {
	_, ok := t.taskEx.RuntimeContext[withItemContextKey]
	if !ok {
		t.taskEx.RuntimeContext[withItemContextKey] = map[string]int64{
			withItemCapacityKey: t.getWithItemConcurrency(),
			withItemCountKey:    actionCount,
		}
	}
}

func (t *WithItemsTask) Run() error {
	if t.taskEx == nil {
		return t.RunNew()
	}
	return t.RunExisting()
}

func (t *WithItemsTask) RunNew() error {
	if t.waiting {
		defer t.deferTask()
		return nil
	}

	taskEx, err := t.createTaskExecution(states.IDLE, "")
	if err != nil {
		return err
	}
	t.taskEx = taskEx

	action, err := t.getAction()
	if err != nil {
		return err
	}

	input, err := t.getActionInput()
	if err != nil {
		return err
	}

	if err := action.ValidateInput(input); err != nil {
		return err
	}

	return action.Schedule(input, t.taskSpec.Target, time.Duration(t.taskSpec.Timeout)*time.Millisecond, 0, "")
}

func (t *WithItemsTask) getActionInputs(withItemValues []objects.Table) ([]objects.Table, error) {
	var inputs []objects.Table

	for i := 0; i < len(withItemValues); i++ {
		itemCtx := withItemValues[i]
		input, err := t.getActionInput()
		if err != nil {
			return nil, err
		}
		for k, v := range itemCtx {
			input[k] = v
		}
		inputs = append(inputs, input)
	}

	return inputs, nil
}

func (t *WithItemsTask) RunExisting() error {
	if t.waiting {
		return nil
	}

	withItemValues, err := t.getWithItemValues()
	if err != nil {
		return err
	}

	inputs, err := t.getActionInputs(withItemValues)
	if err != nil {
		return err
	}

	for i := 0; i < len(inputs); i++ {
		action, err := t.getAction()
		if err != nil {
			return err
		}

		input := inputs[i]

		err = action.ValidateInput(input)
		if err != nil {
			return err
		}

		err = action.Schedule(input, t.taskSpec.Target, time.Duration(t.taskSpec.Timeout)*time.Millisecond, i, "")

		if err != nil {
			return err
		}
	}

	return nil
}

func (t WithItemsTask) ToMap() map[string]interface{} {
	data := t.BaseTask.ToMap()
	data["with-items"] = t.withItems
	return data
}

func (t *WithItemsTask) IsNew() bool {
	if _, ok := t.taskEx.RuntimeContext[withItemContextKey]; ok {
		return false
	}
	return true
}

func (t *WithItemsTask) scheduleActions() {
	withItemValues, err := t.getWithItemValues()
	if err != nil {
		return
	}

	if t.IsNew() {
		actionCount := len(withItemValues)
		t.prepareRuntimeContext(int64(actionCount))
	}

	allInputs, err := t.getActionInputs(withItemValues)
	if err != nil {
		return
	}

	if len(allInputs) == 0 {
		t.Complete(states.SUCCESS, "")
	} else {
		for i, inputs := range allInputs {
			target := t.getTarget(inputs)
			action, err := t.getAction()
			if err != nil {
				return
			}

			err = action.ValidateInput(inputs)
			if err != nil {
				return
			}

			action.Schedule(inputs, target, 0, i, "")
		}
	}
}

func (t *WithItemsTask) getTarget(inputs objects.Table) string {
	if t.taskSpec.Target == "" {
		return ""
	}

	ctxView := data_flow.NewDataContext(
		inputs,
		t.ctx.GetMap(),
		data_flow.GetWorkflowEnvironment(t.wfExec),
		t.wfExec.Context,
		t.wfExec.Input,
	)
	result, err := expressions.EvaluateRecursively(t.taskSpec.Target, ctxView)
	if err != nil {
		return ""
	}
	return result.(string)
}
