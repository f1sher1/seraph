package workflow

import (
	"fmt"
	"reflect"
	"seraph/app/expressions"
	"seraph/app/objects"
	"seraph/app/workflow/data_flow"
	"seraph/app/workflow/interfaces"
	"seraph/app/workflow/states"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"strings"
)

type BaseWorkflowController struct {
	ctx    *contextx.Context
	Tasks  map[string]objects.TaskSpec
	wfEx   *objects.WorkflowExecution
	wfSpec *objects.WorkflowSpec
}

func (c *BaseWorkflowController) getIdleTaskExecutionTasks() ([]interfaces.Task, error) {
	idleTaskExecs, err := objects.QueryTaskExecutions(c.ctx, c.wfEx.ID, nil, states.IDLE, nil, nil)

	if err != nil {
		return nil, err
	}

	var nextTasks []interfaces.Task
	for _, exec := range idleTaskExecs {
		var triggeredBySlice []objects.Table
		if triggeredBy, ok := exec.RuntimeContext["triggered_by"]; ok {
			triggeredBySlice = triggeredBy.([]objects.Table)
		}

		task, err := NewTask(c.ctx, c.wfEx, exec, nil, exec.InContext, triggeredBySlice)
		if err != nil {
			return nil, err
		}
		nextTasks = append(nextTasks, task)
	}
	return nextTasks, nil
}

func (c *BaseWorkflowController) configureIfJoin(task interfaces.Task, taskSpec objects.TaskSpec) {
	if taskSpec.Join != nil {
		task.SetUniqueKey(fmt.Sprintf("join-task-%s-%s", c.wfEx.ID, taskSpec.Name))
		task.SetWait(true)
	}
}

func (c *BaseWorkflowController) AnyCancels() bool {
	taskExs, err := objects.QueryCanceledTaskExecutions(c.ctx, c.wfEx.ID)
	if err != nil {
		return false
	}
	return len(taskExs) > 0
}

type DirectWorkflowController struct {
	BaseWorkflowController
}

func (d *DirectWorkflowController) AllErrorHandled() bool {
	taskExs, err := objects.QueryErrorTaskExecutions(d.ctx, d.wfEx.ID, false)
	if err != nil {
		return false
	}
	return len(taskExs) == 0
}

func (d *DirectWorkflowController) IsErrorHandled(t *objects.TaskExecution) bool {
	return d.wfSpec.HasOnErrorClause(t.Name)
}

func (d *DirectWorkflowController) MayCompleteWorkflow(ex *objects.TaskExecution) bool {
	return states.IsCompleted(ex.State) && !ex.HasNextTasks
}

func (d *DirectWorkflowController) EvaluateWorkflowFinalContext() (map[string]interface{}, error) {
	var finalCtx map[string]interface{}

	taskExs, err := objects.QueryCompleteTaskExecutions(d.ctx, d.wfEx.ID, false)
	if err != nil {
		return nil, err
	}

	for _, taskEx := range taskExs {
		for k, v := range taskEx.InContext {
			finalCtx[k] = v
		}
		for k, v := range taskEx.Published {
			finalCtx[k] = v
		}
	}
	return finalCtx, nil
}

func (d *DirectWorkflowController) FindIndirectlyAffectedTaskExecutions(name string) []*objects.TaskExecution {
	var taskExs []*objects.TaskExecution

	var allJoins objects.SliceString
	for _, t := range d.Tasks {
		if t.Join != nil {
			// 防止数字型的join混入
			if reflect.ValueOf(t.Join).Kind() == reflect.String {
				allJoins = append(allJoins, t.Join.(string))
			}
		}
	}

	// 获取当前任务下所有的task_executions
	var taskExsIndexed map[string]*objects.TaskExecution
	if len(allJoins) > 0 {
		_taskExs, err := objects.QueryTaskExecutions(d.ctx, d.wfEx.ID, nil, nil, allJoins, nil)
		if err != nil {
			log.Warnf(d.ctx, "error happened when search any task executions [wf_execution_id: %s, joins: %s], error: %s", d.wfEx.ID, allJoins, err.Error())
		} else {
			for _, t := range _taskExs {
				taskExsIndexed[t.Name] = t
			}
		}
	}

	visitedNames := objects.SliceString{name}
	// 获取当前任务的下级关联任务
	clauses := d.findOutboundTaskNames(name)
	for {
		if len(clauses) == 0 {
			break
		}

		lastIdx := len(clauses) - 1
		tName := clauses[lastIdx]
		clauses = clauses[:lastIdx]

		if visitedNames.Has(tName) {
			continue
		}

		if _, ok := d.Tasks[tName]; !ok {
			// 去掉engine命令任务
			continue
		}

		if allJoins.Has(tName) {
			if _, ok := taskExsIndexed[tName]; ok {
				taskExs = append(taskExs, taskExsIndexed[tName])
				continue
			}
		}

		// 获取这个任务下级任务名称
		tClauses := d.findOutboundTaskNames(tName)
		clauses = append(clauses, tClauses...)
	}

	return taskExs
}

func (d *DirectWorkflowController) GetLogicTaskState(ex *objects.TaskExecution) *states.LogicState {
	taskSpec := d.Tasks[ex.Name]
	if taskSpec.Join == nil {
		// 这个任务不需要等待上级任务执行结果
		return &states.LogicState{
			State:     ex.State,
			StateInfo: ex.StateInfo,
		}
	}

	joinExpr := taskSpec.Join
	inTaskSpecs := d.findInboundTaskSpecs(taskSpec)
	if len(inTaskSpecs) == 0 {
		return &states.LogicState{State: states.RUNNING}
	}

	taskExs := d.PrepareTaskExecutions(taskSpec)
	// List of InducedState (task_name, task_ex, state, depth, event_name)
	var inducedStates InducedStates

	for _, s := range inTaskSpecs {
		taskEx := taskExs[s.Name]
		iSt := d.getInducedState(s, taskEx, taskSpec, taskExs)
		inducedStates = append(inducedStates, iSt)
	}

	var specCardinality int
	errorsInfo := inducedStates.Count(states.ERROR)
	runningInfo := inducedStates.Count(states.RUNNING)
	totalCount := len(inducedStates)

	switch reflect.ValueOf(joinExpr).Kind() {
	case reflect.Int:
		specCardinality = joinExpr.(int)
	case reflect.String:
		if joinExpr.(string) == "one" {
			specCardinality = 1
		}
	}

	if specCardinality >= 1 {
		if runningInfo.Count >= specCardinality {
			return newLogicState(states.RUNNING, "", inducedStates.GetTriggeredBy(states.RUNNING), 0)
		}

		if errorsInfo.Count > totalCount-specCardinality {
			return newLogicState(states.ERROR, fmt.Sprintf("Failed by tasks: %s", strings.Join(inducedStates.GetTaskNamesByState(states.ERROR), ",")), nil, 0)
		}

		// Calculate how many tasks need to finish triggering this 'join'.
		cardinality := specCardinality - runningInfo.Count
		return newLogicState(states.WAITING, fmt.Sprintf("Blocked by tasks: %s", strings.Join(inducedStates.GetTaskNamesByState(states.WAITING), ",")), nil, cardinality)

	} else if joinExpr.(string) == "all" {
		if runningInfo.Count == totalCount {
			return newLogicState(states.RUNNING, "", inducedStates.GetTriggeredBy(states.RUNNING), 0)
		}

		if errorsInfo.Count > 0 {
			return newLogicState(states.ERROR, fmt.Sprintf("Failed by tasks: %s", strings.Join(inducedStates.GetTaskNamesByState(states.ERROR), ",")), inducedStates.GetTriggeredBy(states.ERROR), 0)
		}

		cardinality := totalCount - runningInfo.Count
		return newLogicState(states.WAITING, fmt.Sprintf("Blocked by tasks: %s", strings.Join(inducedStates.GetTaskNamesByState(states.WAITING), ",")), nil, cardinality)
	} else {
		log.Errorf(d.ctx, "Unexpected join expression: %s", joinExpr)
		return nil
	}
}

func (d *DirectWorkflowController) PrepareTaskExecutions(taskSpec objects.TaskSpec) map[string]*objects.TaskExecution {
	var taskExs map[string]*objects.TaskExecution
	parentNames := d.GetParentTaskSpecNames(taskSpec, 1)

	if len(parentNames) > 0 {
		_taskExs, err := objects.QueryTaskExecutions(d.ctx, d.wfEx.ID, nil, nil, parentNames, nil)
		if err != nil {
			for _, ex := range _taskExs {
				taskExs[ex.Name] = ex
			}
		}
	}

	for _, name := range parentNames {
		if _, ok := taskExs[name]; !ok {
			taskExs[name] = nil
		}
	}

	return taskExs
}

func (d *DirectWorkflowController) GetParentTaskSpecNames(taskSpec objects.TaskSpec, depth int) objects.SliceString {
	var names objects.SliceString
	inTaskSpec := d.findInboundTaskSpecs(taskSpec)
	if len(inTaskSpec) == 0 {
		names = append(names, taskSpec.Name)
		return names
	}

	for _, ts := range inTaskSpec {
		parentTaskSpecNames := d.GetParentTaskSpecNames(ts, depth+1)
		for _, name := range parentTaskSpecNames {
			names = append(names, name)
		}
	}

	if depth > 1 {
		names = append(names, taskSpec.Name)
	}
	return names
}

func (d *DirectWorkflowController) Initialize(spec *objects.WorkflowSpec) error {
	d.wfSpec = spec
	d.Tasks = spec.Tasks
	return nil
}

func (d *DirectWorkflowController) SetWorkflowExecution(wfEx *objects.WorkflowExecution) {
	d.wfEx = wfEx
}

func (d DirectWorkflowController) findOutboundTaskNames(name string) []string {
	var names []string
	task, ok := d.Tasks[name]
	if ok {
		for taskName := range task.GetOnErrorClause() {
			names = append(names, taskName)
		}
		for taskName := range task.GetOnSuccessClause() {
			names = append(names, taskName)
		}
		for taskName := range task.GetOnCompleteClause() {
			names = append(names, taskName)
		}
	}
	return names
}

func (d *DirectWorkflowController) findInboundTaskSpecs(taskSpec objects.TaskSpec) []objects.TaskSpec {
	taskName := taskSpec.Name

	var taskSpecs []objects.TaskSpec
	for name, spec := range d.Tasks {
		if d.transitionExists(name, taskName) {
			taskSpecs = append(taskSpecs, spec)
		}
	}
	return taskSpecs
}

func (d *DirectWorkflowController) findOutboundTaskSpecs(taskSpec objects.TaskSpec) []objects.TaskSpec {
	taskName := taskSpec.Name

	var taskSpecs []objects.TaskSpec
	for name, spec := range d.Tasks {
		if d.transitionExists(taskName, name) {
			taskSpecs = append(taskSpecs, spec)
		}
	}
	return taskSpecs
}

func (d *DirectWorkflowController) hasInboundTransitions(taskSpec objects.TaskSpec) bool {
	return len(d.findInboundTaskSpecs(taskSpec)) > 0
}

func (d *DirectWorkflowController) hasOutboundTransitions(taskSpec objects.TaskSpec) bool {
	return len(d.findOutboundTaskSpecs(taskSpec)) > 0
}

func (d *DirectWorkflowController) transitionExists(fromName string, toName string) bool {
	names := d.findOutboundTaskNames(fromName)
	for _, name := range names {
		if name == toName {
			return true
		}
	}
	return false
}

func (d *DirectWorkflowController) getUpstreamTaskExecutions(taskSpec objects.TaskSpec) []*objects.TaskExecution {
	var taskExecutions []*objects.TaskExecution
	var taskSpecNames []string
	var err error

	for _, spec := range d.findOutboundTaskSpecs(taskSpec) {
		taskSpecNames = append(taskSpecNames, spec.Name)
	}
	if len(taskSpecNames) == 0 {
		return taskExecutions
	}

	if taskSpec.Join == nil {
		// 上级task只有一个
		taskExecutions, err = objects.QueryTaskExecutions(
			d.ctx,
			d.wfEx.ID,
			nil,
			[]string{states.SUCCESS, states.ERROR, states.CANCELLED},
			taskSpecNames[0],
			true,
		)

		if err != nil {
			log.Errorf(d.ctx, "query one upstream task execution failed, error: %s", err.Error())
			return taskExecutions
		}
		return taskExecutions
	}

	// 多个上级task
	taskExecutions, err = objects.QueryTaskExecutions(
		d.ctx,
		nil,
		nil,
		[]string{states.SUCCESS, states.ERROR, states.CANCELLED},
		taskSpecNames,
		nil,
	)

	if err != nil {
		log.Errorf(d.ctx, "query all upstream task execution failed, error: %s", err.Error())
		return taskExecutions
	}

	var resultExs []*objects.TaskExecution

	for _, execution := range taskExecutions {
		if execution.IsInNextTasks(taskSpec.Name) {
			resultExs = append(resultExs, execution)
		}
	}

	return resultExs
}

func (d *DirectWorkflowController) getTaskInboundContext(taskSpec objects.TaskSpec) objects.Table {
	upstreamTaskExecutions := d.getUpstreamTaskExecutions(taskSpec)
	ctx := map[string]interface{}{}

	for _, execution := range upstreamTaskExecutions {
		for k, v := range execution.InContext {
			ctx[k] = v
		}
		for k, v := range execution.Published {
			ctx[k] = v
		}
	}
	return ctx
}

func (d *DirectWorkflowController) GetStartTasks() ([]interfaces.Task, error) {
	var nextTasks []interfaces.Task

	for name := range d.Tasks {
		spec := d.Tasks[name]
		if !d.hasInboundTransitions(spec) {
			task, err := NewTask(d.ctx, d.wfEx, nil, &spec, d.getTaskInboundContext(spec), nil)
			if err != nil {
				return nil, err
			}
			nextTasks = append(nextTasks, task)
		}
	}
	return nextTasks, nil
}

func (d *DirectWorkflowController) GetNextTasksInfo(taskCtx map[string]interface{}, taskExec *objects.TaskExecution) ([]interface{}, error) {
	var tasksInfo []interface{}
	taskName := taskExec.Execution.Name
	state := taskExec.Execution.State

	dataCtx := data_flow.NewDataContext(
		data_flow.GetTaskInfo(taskExec),
		taskCtx,
		data_flow.GetWorkflowEnvironment(d.wfEx),
		d.wfEx.Context,
		d.wfEx.Input,
	)

	if states.IsErrored(state) {
		for toTaskName, toConditionStr := range d.Tasks[taskName].GetOnErrorClause() {
			if toConditionStr == "" {
				tasksInfo = append(tasksInfo, []interface{}{toTaskName, "on-error"})
			} else if result, err := expressions.Evaluate(toConditionStr, dataCtx); err != nil && result.(bool) {
				tasksInfo = append(tasksInfo, []interface{}{toTaskName, "on-error"})
			}
		}
	} else if states.IsSuccess(state) {
		for toTaskName, toConditionStr := range d.Tasks[taskName].GetOnSuccessClause() {
			if toConditionStr == "" {
				tasksInfo = append(tasksInfo, []interface{}{toTaskName, "on-success"})
			} else if result, err := expressions.Evaluate(toConditionStr, dataCtx); err != nil && result.(bool) {
				tasksInfo = append(tasksInfo, []interface{}{toTaskName, "on-success"})
			}
		}
	}

	if states.IsCompleted(state) && !states.IsCanceled(state) {
		for toTaskName, toConditionStr := range d.Tasks[taskName].GetOnCompleteClause() {
			if toConditionStr == "" {
				tasksInfo = append(tasksInfo, []interface{}{toTaskName, "on-complete"})
			} else if result, err := expressions.Evaluate(toConditionStr, dataCtx); err != nil && result.(bool) {
				tasksInfo = append(tasksInfo, []interface{}{toTaskName, "on-complete"})
			}
		}
	}

	return tasksInfo, nil
}

func (d *DirectWorkflowController) GetNextTasksForTask(taskEx *objects.TaskExecution) ([]interfaces.Task, error) {
	var nextTasks []interfaces.Task

	taskCtx := data_flow.EvaluateTaskOutBondContext(taskEx)

	nextTasksInfo, err := d.GetNextTasksInfo(taskCtx, taskEx)
	if err != nil {
		return nil, err
	}

	_spec, err := d.wfEx.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("can't parse spec for workflow, error: %s", err.Error())
	}

	var newEngineTask NewEngineTaskFunc
	for _, _taskCond := range nextTasksInfo {
		var task interfaces.Task
		taskCond := _taskCond.([]interface{})
		taskName := taskCond[0].(string)
		eventName := taskCond[1].(string)

		taskSpec, ok := d.Tasks[taskName]

		if !ok {
			// 可能是内置命令
			if !IsEngineTask(taskName) {
				return nil, fmt.Errorf("task '%s' missing", taskName)
			} else {
				newEngineTask = GetEngineTask(taskName)
			}
		}

		triggeredBy := []objects.Table{
			{
				"task_id": taskEx.ID,
				"event":   eventName,
			},
		}

		if taskName == "" {
			taskSpec = d.Tasks[taskEx.Name]
		}

		if newEngineTask != nil {
			task = newEngineTask(d.ctx, d.wfEx, _spec, &taskSpec, taskCtx)
		} else {
			task, err = NewTask(d.ctx, d.wfEx, nil, &taskSpec, taskCtx, triggeredBy)
			if err != nil {
				return nil, err
			}
			d.configureIfJoin(task, taskSpec)
		}

		newEngineTask = nil
		nextTasks = append(nextTasks, task)
	}

	return nextTasks, nil
}

func (d *DirectWorkflowController) GetNextTasks(taskExec *objects.TaskExecution) ([]interfaces.Task, error) {
	if states.IsCompleted(d.wfEx.State) {
		return []interfaces.Task{}, nil
	}

	// 添加上所有上次未执行完的任务
	nextTasks, err := d.getIdleTaskExecutionTasks()
	if err != nil {
		return nil, err
	}

	var taskExecs []*objects.TaskExecution
	if taskExec == nil {
		allTaskExecs, err := d.wfEx.GetChildrenTaskExecutions()
		if err != nil {
			return nil, err
		}

		if len(allTaskExecs) == 0 {
			// 没有任何关联任务，说明是第一次执行
			return d.GetStartTasks()
		}

		for _, exec := range allTaskExecs {
			if states.IsCompleted(exec.State) && !exec.Processed {
				// 只加入未开始执行的任务
				taskExecs = append(taskExecs, exec)
			}
		}
	} else {
		taskExecs = append(taskExecs, taskExec)
	}

	for _, exec := range taskExecs {
		_nextTasks, err := d.GetNextTasksForTask(exec)
		if err != nil {
			return nil, err
		}
		nextTasks = append(nextTasks, _nextTasks...)
	}

	return nextTasks, nil
}

func (d *DirectWorkflowController) getInducedState(inTaskSpec objects.TaskSpec, inTaskEx *objects.TaskExecution, joinTaskSpec objects.TaskSpec, taskExsCache map[string]*objects.TaskExecution) *InducedState {
	joinTaskSpecName := joinTaskSpec.Name
	if inTaskEx == nil {
		possible, depth := d.getPossibleRoute(inTaskSpec, taskExsCache, 1)
		if possible {
			return &InducedState{
				TaskName:  inTaskSpec.Name,
				TaskEx:    inTaskEx,
				State:     states.WAITING,
				Depth:     depth,
				EventName: "",
			}
		} else {
			return &InducedState{
				TaskName:  inTaskSpec.Name,
				TaskEx:    inTaskEx,
				State:     states.ERROR,
				Depth:     depth,
				EventName: "impossible route",
			}
		}
	}

	if !states.IsCompleted(inTaskEx.State) {
		return &InducedState{
			TaskName:  inTaskSpec.Name,
			TaskEx:    inTaskEx,
			State:     states.WAITING,
			Depth:     1,
			EventName: "",
		}
	}

	var nextTasksIndexed map[string]string
	// [(task name, event name), ...]
	nextTasks := inTaskEx.GetNextTasks()
	for _, t := range nextTasks {
		nextTasksIndexed[t[0]] = t[1]
	}

	eventName, ok := nextTasksIndexed[joinTaskSpecName]
	if !ok {
		return &InducedState{
			TaskName:  inTaskSpec.Name,
			TaskEx:    inTaskEx,
			State:     states.ERROR,
			Depth:     1,
			EventName: "not triggered",
		}
	}

	return &InducedState{
		TaskName:  inTaskSpec.Name,
		TaskEx:    inTaskEx,
		State:     states.RUNNING,
		Depth:     1,
		EventName: eventName,
	}
}

func (d *DirectWorkflowController) getPossibleRoute(spec objects.TaskSpec, taskExsCache map[string]*objects.TaskExecution, searchDepth int) (bool, int) {
	inTaskSpecs := d.findInboundTaskSpecs(spec)
	if len(inTaskSpecs) == 0 {
		return true, searchDepth
	}
	for _, tSpec := range inTaskSpecs {
		if _, ok := taskExsCache[tSpec.Name]; !ok {
			tTaskExs := d.PrepareTaskExecutions(spec)
			for name, ex := range tTaskExs {
				taskExsCache[name] = ex
			}
		}

		tEx := taskExsCache[tSpec.Name]
		if tEx == nil {
			possible, depth := d.getPossibleRoute(tSpec, taskExsCache, searchDepth+1)
			if possible {
				return true, depth
			}
		} else {
			if !states.IsCompleted(tEx.State) {
				return true, searchDepth
			}

			if tEx.IsInNextTasks(spec.Name) {
				return true, searchDepth
			}
		}
	}
	return false, searchDepth
}

type InducedState struct {
	TaskName  string
	TaskEx    *objects.TaskExecution
	State     string
	Depth     int
	EventName string
}

type InducedStates []*InducedState

type InducedStateCountInfo struct {
	Count int
	Depth int
}

func (s InducedStates) Count(state string) *InducedStateCountInfo {
	info := &InducedStateCountInfo{}
	for _, st := range s {
		if st.State == state {
			info.Count += 1
			info.Depth += st.Depth
		}
	}
	return info
}

func (s InducedStates) GetTriggeredBy(state string) []objects.Table {
	var triggeredBy []objects.Table
	for _, st := range s {
		if st.State == state {
			triggeredBy = append(triggeredBy, objects.Table{
				"task_id": st.TaskEx.ID,
				"event":   st.EventName,
			})
		}
	}
	return triggeredBy
}

func (s InducedStates) GetTaskNamesByState(state string) []string {
	var names []string
	for _, st := range s {
		if st.State == state {
			names = append(names, st.TaskName)
		}
	}
	return names
}

func InitializeFlowController(ctx *contextx.Context, wfSpec *objects.WorkflowSpec) (interfaces.WorkflowController, error) {
	var flowController interfaces.WorkflowController
	switch wfSpec.Type {
	case "direct":
		flowController = &DirectWorkflowController{BaseWorkflowController{ctx: ctx}}
	case "reverse":
		//w.flowController = &ReverseWorkflowController{}
	default:
		return nil, fmt.Errorf("invalid workflow type '%s'", wfSpec.Type)
	}
	err := flowController.Initialize(wfSpec)
	if err != nil {
		return nil, err
	}
	return flowController, nil
}

func GetFlowController(wfEx *objects.WorkflowExecution, wfSpec *objects.WorkflowSpec) (interfaces.WorkflowController, error) {
	var err error
	if wfSpec == nil {
		wfSpec, err = wfEx.GetSpec()
		if err != nil {
			return nil, err
		}
	}
	return InitializeFlowController(wfEx.GetContext(), wfSpec)
}

func newLogicState(state string, stateInfo string, triggeredBy []objects.Table, cardinality int) *states.LogicState {
	ls := &states.LogicState{
		State:       state,
		TriggeredBy: []map[string]interface{}{},
		StateInfo:   stateInfo,
		Cardinality: cardinality,
	}
	if triggeredBy != nil && len(triggeredBy) > 0 {
		for _, _t := range triggeredBy {
			t := map[string]interface{}{}
			for k, v := range _t {
				t[k] = v
			}
			ls.TriggeredBy = append(ls.TriggeredBy, t)
		}
	}
	return ls
}
