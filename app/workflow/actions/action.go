package actions

import (
	"fmt"
	"seraph/app/config"
	"seraph/app/engine/client"
	"seraph/app/executor"
	"seraph/app/expressions"
	"seraph/app/objects"
	"seraph/app/workflow/data_flow"
	"seraph/app/workflow/states"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"strings"
	"time"
)

type TaskBaseAction struct {
	ctx        *contextx.Context
	taskEx     *objects.TaskExecution
	actionEx   *objects.ActionExecution
	actionSpec *objects.ActionSpec
	actionDef  *objects.ActionDefinition
}

func (t *TaskBaseAction) GetTaskExecution() *objects.TaskExecution {
	return t.taskEx
}

func (t *TaskBaseAction) GetTaskExecutionID() string {
	if t.taskEx != nil {
		return t.taskEx.ID
	}
	return ""
}

func (t *TaskBaseAction) prepareExecutionContext() objects.Table {
	ctx := objects.Table{}
	if t.taskEx != nil {
		ctx["task_execution_id"] = t.taskEx.ID

		wfEx, err := t.taskEx.GetWorkflowExecution()
		if err != nil {
			log.Debugf(t.ctx, "prepare execution context failed, error: %s", err.Error())
		} else {
			ctx["workflow_execution_id"] = wfEx.ID
			ctx["workflow_name"] = wfEx.Name
		}
	}

	if t.actionEx != nil {
		ctx["action_execution_id"] = t.actionEx.ID
	}
	return ctx
}

func (t *TaskBaseAction) createActionExecution(input objects.Table, runtimeContext objects.Table, description, id string) error {
	ex := objects.NewActionExecution()
	ex.ID = id
	ex.Inputs = map[string]interface{}(input)
	ex.RuntimeContext = map[string]interface{}(runtimeContext)
	ex.Description = description
	if t.actionSpec != nil {
		ex.Name = t.actionSpec.Name
		ex.Spec = t.actionSpec.ToString()
	} else if t.actionDef != nil {
		ex.Name = t.actionDef.Name
	}
	ex.State = states.RUNNING
	if t.taskEx != nil {
		ex.TaskExecutionID = t.taskEx.ID
		ex.WorkflowID = t.taskEx.WorkflowID
		ex.WorkflowName = t.taskEx.WorkflowName
		ex.WorkflowNamespace = t.taskEx.WorkflowNamespace
		ex.ProjectID = t.taskEx.ProjectID
	} else {
		ex.ProjectID = t.ctx.GetProjectID()
	}
	err := ex.Save(t.ctx)

	t.actionEx = ex
	return err
}

type WorkflowAction struct {
	TaskBaseAction
	wfName string
	wfEx   *objects.WorkflowExecution
}

func (c *WorkflowAction) Fail(msg string) {
	c.wfEx.State = states.ERROR
	c.wfEx.Output = map[string]interface{}{
		"result": msg,
	}
	c.wfEx.FinishedAt = time.Now().UTC()
	// c.wfEx.Save(nil)
	c.wfEx.Update(nil, "State", "Output", "FinishedAt")
}

func (c WorkflowAction) Complete(result *objects.ActionResult) error {
	// No-op because in case of workflow result is already processed.
	return nil
}

func (c *WorkflowAction) Initialize() error {
	taskEx, err := c.wfEx.GetTaskExecution()
	if err != nil {
		return err
	}
	c.taskEx = taskEx
	return nil
}

func (c WorkflowAction) ValidateInput(input objects.Table) error {
	return nil
}

func (c *WorkflowAction) Schedule(input objects.Table, target string, timeout time.Duration, index int, description string) error {
	parentWfEx, err := c.taskEx.GetWorkflowExecution()
	if err != nil {
		return err
	}

	parentWfSpec, err := parentWfEx.GetSpec()
	if err != nil {
		return err
	}

	parentWfNamespace := parentWfEx.Params["namespace"]
	wfDef, err := ResolveWorkflowDefinition(c.ctx, parentWfEx.WorkflowName, parentWfSpec.Name, parentWfNamespace.(string), c.wfName)
	if err != nil {
		return err
	}

	wfSpec, err := wfDef.GetSpec()
	if err != nil {
		return err
	}

	rootExeID := parentWfEx.RootExecutionID
	if rootExeID == "" {
		rootExeID = parentWfEx.ID
	}

	wfParams := map[string]interface{}{
		"root_execution_id": rootExeID,
		"task_execution_id": c.taskEx.ID,
		"index":             index,
		"namespace":         parentWfNamespace,
	}

	if notifyTo, ok := parentWfEx.Params["namespace"]; ok {
		wfParams["notify"] = notifyTo
	}

	removedNames := []string{}
	for k, v := range input {
		found := false
		for _, name := range wfSpec.GetInputNames() {
			if k == name {
				found = true
				break
			}
		}
		if !found {
			wfParams[k] = v
			removedNames = append(removedNames, k)
		}
	}

	for _, name := range removedNames {
		delete(input, name)
	}

	// 通过RPC启动
	engineClient, err := client.NewClient()
	if err != nil {
		return err
	}
	wfEx, err := engineClient.StartWorkflow(c.ctx, wfDef.ID, wfDef.Namespace, "", input, "sub-workflow execution", wfParams)
	if err != nil {
		return err
	}
	c.wfEx = wfEx
	return nil
}

func ResolveWorkflowDefinition(ctx *contextx.Context, parentWfName, parentWfSpecName, namespace, wfSpecName string) (*objects.WorkflowDefinition, error) {
	var wfDef *objects.WorkflowDefinition

	if parentWfName != parentWfSpecName {
		// 工作流属于一个workbook
		wbName := strings.TrimRight(parentWfName, parentWfSpecName)
		wbName = wbName[:len(wbName)-2]

		wfName := fmt.Sprintf("%s.%s", wbName, wfSpecName)
		if _definitions, err := objects.QueryWorkflowDefinitions(ctx, wfName, namespace); err != nil {
			return nil, err
		} else {
			wfDef = _definitions[0]
		}
	}

	if wfDef == nil {
		if _definitions, err := objects.QueryWorkflowDefinitions(ctx, wfSpecName, namespace); err != nil {
			return nil, err
		} else {
			wfDef = _definitions[0]
		}
	}

	if wfDef == nil {
		return nil, fmt.Errorf("failed to find workflow [action_name=%s] [namespace=%s]", wfSpecName, namespace)
	}

	return wfDef, nil
}

func ResolveActionDefinition(ctx *contextx.Context, specName, wfName, wfSpecName string) (*objects.ActionDefinition, error) {
	var actionDef *objects.ActionDefinition
	var err error

	if wfName != "" && wfName != wfSpecName {
		// 工作流属于一个workbook
		wbName := strings.TrimRight(wfName, wfSpecName)
		wbName = wbName[:len(wbName)-2]

		actionName := fmt.Sprintf("%s.%s", wbName, specName)
		actionDef, err = objects.QueryActionDefinitionByName(ctx, actionName)
		if err != nil {
			return nil, err
		}
	}

	if actionDef == nil {
		actionDef, err = objects.QueryActionDefinitionByName(ctx, specName)
		if err != nil {
			return nil, err
		}
	}

	if actionDef == nil {
		return nil, fmt.Errorf("failed to find action [action_name=%s]", specName)
	}

	return actionDef, err
}

func NewWorkflowAction(name string, taskEx *objects.TaskExecution, wfEx *objects.WorkflowExecution) (*WorkflowAction, error) {
	action := &WorkflowAction{}
	action.taskEx = taskEx
	action.wfName = name
	action.wfEx = wfEx
	return action, action.Initialize()
}

type CommandAction struct {
	TaskBaseAction
	taskCtx objects.Table
	wfCtx   objects.Table
}

func (c *CommandAction) Fail(msg string) {
	c.actionEx.State = states.ERROR
	c.actionEx.Outputs = map[string]interface{}{
		"result": msg,
	}
	c.actionEx.FinishedAt = time.Now().UTC()
	// c.actionEx.Save(nil)
	c.actionEx.Update(c.ctx, "State", "Outputs", "FinishedAt")
}

func (c *CommandAction) fillState(result *objects.ActionResult) {
	if result.IsSuccess() {
		c.actionEx.State = states.SUCCESS
	} else if result.IsCancel() {
		c.actionEx.State = states.CANCELLED
	} else {
		c.actionEx.State = states.ERROR
	}
}

func (c *CommandAction) Complete(result *objects.ActionResult) error {
	if states.IsCompleted(c.actionEx.State) {
		log.Warnf(c.ctx, "action execution %s is %s, skip complete it", c.actionEx.ID)
		return nil
	}

	prevState := c.actionEx.State
	c.fillState(result)
	c.actionEx.Outputs = result.ToMap()
	c.actionEx.FinishedAt = time.Now().UTC()
	// c.actionEx.Save(nil)
	c.actionEx.Update(c.ctx, "State", "Outputs", "FinishedAt")

	if prevState != c.actionEx.State {
		log.Debugf(c.ctx, "Action '%s' (%s)(task=%s) [%s -> %s, %s]", c.actionEx.Name, c.actionEx.ID, c.GetTaskExecutionID(), prevState, c.actionEx.State, result)
	}

	return nil
}

type AdHocAction struct {
	CommandAction
	taskEx         *objects.TaskExecution
	baseActionDefs []*objects.ActionDefinition
	preparedInput  map[string]interface{}
	prepared       bool
}

func NewAdHocAction(ctx *contextx.Context, actionDef *objects.ActionDefinition, taskEx *objects.TaskExecution, taskCtx objects.Table, wfCtx objects.Table) (*AdHocAction, error) {
	action := &AdHocAction{}
	action.ctx = ctx
	action.taskEx = taskEx
	action.actionDef = actionDef
	action.taskCtx = taskCtx
	action.wfCtx = wfCtx
	action.actionSpec = actionDef.GetSpec()
	return action, action.Initialize()
}

func (c *AdHocAction) gatherBaseActions(actionDef *objects.ActionDefinition, baseActionDef *objects.ActionDefinition) (*objects.ActionDefinition, error) {
	var err error

	c.baseActionDefs = append(c.baseActionDefs, actionDef)

	originBaseName := c.actionSpec.Name
	actionNames := objects.SliceString{originBaseName}

	base := baseActionDef
	for {
		if actionNames.Has(base.Name) {
			break
		}

		actionNames = append(actionNames, base.Name)
		c.baseActionDefs = append(c.baseActionDefs, base)

		base, err = objects.QueryActionDefinitionByName(c.ctx, base.GetSpec().GetBase())
		if err != nil {
			return nil, err
		}
	}

	return base, nil
}

func (c *AdHocAction) Initialize() error {
	baseActionDef, err := objects.QueryActionDefinitionByName(c.ctx, c.actionSpec.GetBase())
	if err != nil {
		return err
	}
	if baseActionDef == nil {
		return fmt.Errorf("failed to find action [action_name=%s]", c.actionSpec.GetBase())
	}

	baseActionDef, err = c.gatherBaseActions(c.actionDef, baseActionDef)
	if err != nil {
		return err
	}

	c.actionDef = baseActionDef
	return nil
}

func (c *AdHocAction) prepareInput(input objects.Table) (map[string]interface{}, error) {
	if c.prepared {
		return c.preparedInput, nil
	}

	c.preparedInput = map[string]interface{}{}

	_preparedInput := input
	var wfEx *objects.WorkflowExecution
	var err error
	if c.taskEx != nil {
		wfEx, err = c.taskEx.GetWorkflowExecution()
		if err != nil {
			return nil, err
		}
	}

	for _, def := range c.baseActionDefs {
		_preparedInput = def.GetMergedInput(_preparedInput)

		actSpec := def.GetSpec()
		if actSpec != nil {
			_preparedInput = actSpec.GetMergedInput(_preparedInput)
		}

		baseInputExpr := actSpec.GetBaseInput()
		if baseInputExpr != nil && len(baseInputExpr) > 0 {
			ctx := data_flow.NewDataContext(
				_preparedInput,
				c.taskCtx,
				data_flow.GetWorkflowEnvironment(wfEx),
				c.wfCtx,
			)

			__preparedInput, err := expressions.EvaluateRecursively(_preparedInput, ctx)
			if err != nil {
				return nil, err
			}

			_preparedInput = __preparedInput.(map[string]interface{})
		} else {
			_preparedInput = map[string]interface{}{}
		}
	}

	c.preparedInput = _preparedInput
	c.prepared = true
	//
	//ctx := data_flow.NewDataContext(
	//	input,
	//	c.taskCtx,
	//	data_flow.GetWorkflowEnvironment(wfEx),
	//	c.wfCtx,
	//	)
	//preparedInput, err := expressions.EvaluateRecursively(_preparedInput, ctx)
	//if err != nil {
	//	return err
	//}
	//for name, value := range preparedInput.(map[string]interface{}) {
	//	c.preparedInput[name] = value
	//}
	//
	//expectedInputNames := objects.SliceString(c.actionSpec.GetInputNames())
	//for _, name := range expectedInputNames {
	//	if !input.Has(name) {
	//		return fmt.Errorf("input name %s is missing", name)
	//	}
	//}
	return c.preparedInput, nil
}

func (c *AdHocAction) ValidateInput(input objects.Table) error {
	expectedInputNames := objects.SliceString(c.actionSpec.GetInputNames())
	for _, name := range expectedInputNames {
		if !input.Has(name) {
			return fmt.Errorf("input name %s is missing", name)
		}
	}
	return nil
}

func (c *AdHocAction) schedule(ctx *contextx.Context, preparedInput, exCtx objects.Table, target string, timeout time.Duration) {
	time.Sleep(100 * time.Millisecond)
	actExecutor := executor.GetExecutor(config.Config.Executor.Kind)
	actExecutor.RunAction(ctx, c.actionEx, c.actionEx.ID, c.actionDef.ActionClass, objects.Table(c.actionDef.Attributes), preparedInput, exCtx, target, timeout)
}

func (c *AdHocAction) Schedule(input objects.Table, target string, timeout time.Duration, index int, description string) error {
	runtimeCtx := objects.Table{
		"index": index,
	}

	prepareInput, err := c.prepareInput(input)
	if err != nil {
		return err
	}

	err = c.createActionExecution(prepareInput, runtimeCtx, description, "")
	if err != nil {
		return err
	}

	exCtx := c.prepareExecutionContext()

	go c.schedule(c.ctx.Clone(), input, exCtx, target, timeout)
	return nil
}

func (c *AdHocAction) Complete(result *objects.ActionResult) error {
	if states.IsCompleted(c.actionEx.State) {
		log.Warnf(c.ctx, "action execution %s is %s, skip complete it", c.actionEx.ID)
		return nil
	}

	prevState := c.actionEx.State
	c.fillState(result)

	if !result.IsError() {
		for i := len(c.baseActionDefs) - 1; i >= 0; i-- {
			actDef := c.baseActionDefs[i]
			actSpec := actDef.GetSpec()

			newData, err := expressions.EvaluateRecursively(actSpec.Output, result.Data.(map[string]interface{}))
			if err != nil {
				return err
			}
			result.Data = newData
		}
		c.actionEx.Outputs = result.ToMap()
	} else {
		c.actionEx.Outputs = result.ToMap()
	}
	c.actionEx.FinishedAt = time.Now().UTC()

	// c.actionEx.Save(nil)
	c.actionEx.Update(c.ctx, "State", "Outputs", "FinishedAt")

	if prevState != c.actionEx.State {
		log.Debugf(c.ctx, "Action '%s' (%s)(task=%s) [%s -> %s, %s]", c.actionEx.Name, c.actionEx.ID, c.GetTaskExecutionID(), prevState, c.actionEx.State, result)
	}

	return nil
}

type GolangAction struct {
	CommandAction
}

func (c *GolangAction) Initialize() error {
	return nil
}

func (c GolangAction) ValidateInput(input objects.Table) error {
	return nil
}

func (c *GolangAction) schedule(ctx *contextx.Context, preparedInput, exCtx objects.Table, target string, timeout time.Duration) {
	time.Sleep(100 * time.Millisecond)
	actExecutor := executor.GetExecutor(config.Config.Executor.Kind)
	log.Debugf(ctx, "Running %#v", c.taskEx.Name)
	actExecutor.RunAction(ctx, c.actionEx, c.actionEx.ID, c.actionDef.ActionClass, objects.Table(c.actionDef.Attributes), preparedInput, exCtx, target, timeout)
}

func (c *GolangAction) Schedule(input objects.Table, target string, timeout time.Duration, index int, description string) error {
	runtimeCtx := objects.Table{
		"index": index,
	}

	// log.Errorf("#5.0 %#v", input)
	_preparedInput := c.actionDef.GetMergedInput(input)

	err := c.createActionExecution(_preparedInput, runtimeCtx, description, "")
	if err != nil {
		return err
	}

	exCtx := c.prepareExecutionContext()
	// actEx, _ := objects.QueryActionExecutionByID(c.ctx, c.actionEx.ID)
	go c.schedule(c.ctx.Clone(), _preparedInput, exCtx, target, timeout)
	return nil
}

func NewGolangAction(ctx *contextx.Context, actionDef *objects.ActionDefinition, actionEx *objects.ActionExecution, taskEx *objects.TaskExecution) (*GolangAction, error) {
	action := &GolangAction{}
	action.ctx = ctx
	action.actionEx = actionEx
	action.actionDef = actionDef
	action.taskEx = taskEx
	return action, action.Initialize()
}
