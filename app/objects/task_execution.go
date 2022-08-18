package objects

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"seraph/app/db/models"
	"seraph/app/workflow/states"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

type TaskExecution struct {
	*models.TaskExecution
	ContextObject
	PersistentObject
}

func (e *TaskExecution) Save(ctx *contextx.Context) error {
	if !e.IsCreated() {
		e.CreatedAt = time.Now().UTC()
		if e.ID == "" {
			e.ID = uuid.NewString()
		}
		e.UpdatedAt = e.CreatedAt
	} else {
		e.UpdatedAt = time.Now().UTC()
	}

	taskExModel := e.GetQuery(ctx).TaskExecution
	err := taskExModel.WithContext(context.Background()).Save(e.TaskExecution)
	if err != nil {
		return err
	}
	e.SetContext(ctx)
	e.SetCreated()
	return nil
}

func (e *TaskExecution) Update(ctx *contextx.Context, fields ...string) error {
	taskExModel := e.GetQuery(ctx).TaskExecution
	var flds []field.Expr
	for _, f := range fields {
		switch f {
		case "Deleted":
			flds = append(flds, taskExModel.Deleted)
		case "DeletedAt":
			flds = append(flds, taskExModel.DeletedAt)
		case "FinishedAt":
			flds = append(flds, taskExModel.FinishedAt)
		case "State":
			flds = append(flds, taskExModel.State)
		case "StateInfo":
			flds = append(flds, taskExModel.StateInfo)
		case "Processed":
			flds = append(flds, taskExModel.Processed)
		case "NextTasks":
			flds = append(flds, taskExModel.NextTasks)
		case "HasNextTasks":
			flds = append(flds, taskExModel.HasNextTasks)
		case "ErrorHandled":
			flds = append(flds, taskExModel.ErrorHandled)
		}
	}
	result, err := taskExModel.WithContext(context.Background()).Select(flds...).Where(taskExModel.ID.Eq(e.TaskExecution.ID)).Updates(e.TaskExecution)
	if err != nil {
		log.Errorf(ctx, "Save data error: %v", err.Error())
		return err
	}
	log.Debugf(ctx, "Update data result %#v", result)
	return nil
}

func (e *TaskExecution) Delete(ctx *contextx.Context) error {
	if !e.IsCreated() {
		return fmt.Errorf("object %s isn't a persistent object, can't delete it", e.ID)
	}

	e.Deleted = 1
	e.DeletedAt = time.Now().UTC()
	// return e.Save(ctx)
	return e.Update(ctx, "Deleted", "DeletedAt")
}

func (e *TaskExecution) GetWorkflowExecution() (*WorkflowExecution, error) {
	wfEx, err := QueryWorkflowExecutionByID(e.GetContext(), e.WorkflowExecutionID)
	if err != nil {
		return nil, err
	}
	return wfEx, nil
}

func (e *TaskExecution) GetSpec() (*TaskSpec, error) {
	spec := &TaskSpec{}
	if err := json.Unmarshal([]byte(e.Spec), &spec); err != nil {
		return nil, err
	}
	if err := spec.Initialize(); err != nil {
		return nil, err
	}
	return spec, nil
}

func (e *TaskExecution) IsInNextTasks(name string) bool {
	for _, _taskCond := range e.NextTasks {
		taskCond := _taskCond.([]string)
		if taskCond[0] == name {
			return true
		}
	}
	return false
}

func (e *TaskExecution) GetNextTasks() [][]string {
	var nTasks [][]string
	for _, _taskCond := range e.NextTasks {
		taskCond := _taskCond.([]string)
		nTasks = append(nTasks, taskCond)
	}
	return nTasks
}

func (e *TaskExecution) GetActionExecutions() ([]*ActionExecution, error) {
	return QueryActionExecutionsByTaskID(e.GetContext(), e.ID)
}

func NewTaskExecution() *TaskExecution {
	return &TaskExecution{
		TaskExecution: &models.TaskExecution{},
	}
}

func NewTaskExecutionFromDB(ctx *contextx.Context, ex *models.TaskExecution) *TaskExecution {
	if ex == nil {
		return nil
	}
	taskEx := &TaskExecution{
		TaskExecution: ex,
	}
	taskEx.SetContext(ctx)
	taskEx.SetCreated()
	return taskEx
}

func QueryTaskExecutions(ctx *contextx.Context, wfExID interface{}, uniqueKey interface{}, state interface{}, name interface{}, processed interface{}) ([]*TaskExecution, error) {
	var taskExs []*TaskExecution
	var conditions []gen.Condition

	taskExModel := GetQuery(ctx).TaskExecution
	projectId := ctx.GetProjectID()
	conditions = append(conditions, taskExModel.Deleted.Eq(0))

	if projectId != "" {
		conditions = append(conditions, taskExModel.ProjectID.Eq(projectId))
	}
	if uniqueKey != nil {
		conditions = append(conditions, taskExModel.UniqueKey.Eq(uniqueKey.(string)))
	}
	if wfExID != nil {
		conditions = append(conditions, taskExModel.WorkflowExecutionID.Eq(wfExID.(string)))
	}
	if state != nil {
		switch reflect.ValueOf(state).Kind() {
		case reflect.String:
			conditions = append(conditions, taskExModel.State.Eq(state.(string)))
		case reflect.Slice:
			conditions = append(conditions, taskExModel.State.In(state.([]string)...))
		}
	}
	if name != nil {
		switch reflect.ValueOf(name).Kind() {
		case reflect.String:
			conditions = append(conditions, taskExModel.Name.Eq(name.(string)))
		case reflect.Slice:
			conditions = append(conditions, taskExModel.Name.In(name.([]string)...))
		}
	}
	if processed != nil {
		conditions = append(conditions, taskExModel.Processed.Is(processed.(bool)))
	}

	_taskExs, err := taskExModel.WithContext(context.Background()).Where(conditions...).Find()
	if err != nil {
		return nil, err
	}

	for _, ex := range _taskExs {
		taskEx := NewTaskExecutionFromDB(ctx, ex)
		taskExs = append(taskExs, taskEx)
	}
	return taskExs, err
}

func QueryCompleteTaskExecutions(ctx *contextx.Context, wfExID string, hasNextTasks interface{}) ([]*TaskExecution, error) {
	var taskExs []*TaskExecution
	var conditions []gen.Condition

	taskExModel := GetQuery(ctx).TaskExecution
	projectId := ctx.GetProjectID()
	conditions = append(conditions, taskExModel.Deleted.Eq(0))
	conditions = append(conditions, taskExModel.WorkflowExecutionID.Eq(wfExID))
	conditions = append(conditions, taskExModel.State.In(states.CompleteStates...))

	if projectId != "" {
		conditions = append(conditions, taskExModel.ProjectID.Eq(projectId))
	}
	if hasNextTasks != nil {
		conditions = append(conditions, taskExModel.HasNextTasks.Is(hasNextTasks.(bool)))
	}
	_taskExs, err := taskExModel.WithContext(context.Background()).Where(conditions...).Find()
	if err != nil {
		return nil, err
	}

	for _, ex := range _taskExs {
		taskEx := NewTaskExecutionFromDB(ctx, ex)
		taskExs = append(taskExs, taskEx)
	}
	return taskExs, err
}

func QueryTaskExecutionByID(ctx *contextx.Context, id string) (*TaskExecution, error) {
	var conditions []gen.Condition

	taskExModel := GetQuery(ctx).TaskExecution
	projectId := ctx.GetProjectID()
	conditions = append(conditions, taskExModel.Deleted.Eq(0))
	conditions = append(conditions, taskExModel.ID.Eq(id))

	if projectId != "" {
		conditions = append(conditions, taskExModel.ProjectID.Eq(projectId))
	}
	_taskEx, err := taskExModel.WithContext(context.Background()).Where(conditions...).First()
	if IsNotFoundError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if _taskEx != nil {
		taskEx := NewTaskExecutionFromDB(ctx, _taskEx)
		return taskEx, nil
	}
	return nil, err
}

func QueryIncompleteTaskExecutions(ctx *contextx.Context, wfExID string) ([]*TaskExecution, error) {
	var taskExs []*TaskExecution
	var conditions []gen.Condition

	taskExModel := GetQuery(ctx).TaskExecution
	projectId := ctx.GetProjectID()
	conditions = append(conditions, taskExModel.Deleted.Eq(0))
	conditions = append(conditions, taskExModel.WorkflowExecutionID.Eq(wfExID))
	conditions = append(conditions, taskExModel.State.In(states.IncompleteStates...))

	if projectId != "" {
		conditions = append(conditions, taskExModel.ProjectID.Eq(projectId))
	}
	_taskExs, err := taskExModel.WithContext(context.Background()).Where(conditions...).Find()
	if err != nil {
		return nil, err
	}

	for _, ex := range _taskExs {
		taskEx := NewTaskExecutionFromDB(ctx, ex)
		taskExs = append(taskExs, taskEx)
	}
	return taskExs, err
}

func QueryErrorTaskExecutions(ctx *contextx.Context, wfExID string, errorHandled interface{}) ([]*TaskExecution, error) {
	var taskExs []*TaskExecution
	var conditions []gen.Condition

	taskExModel := GetQuery(ctx).TaskExecution
	projectId := ctx.GetProjectID()
	conditions = append(conditions, taskExModel.Deleted.Eq(0))
	conditions = append(conditions, taskExModel.WorkflowExecutionID.Eq(wfExID))
	conditions = append(conditions, taskExModel.State.Eq(states.ERROR))

	if projectId != "" {
		conditions = append(conditions, taskExModel.ProjectID.Eq(projectId))
	}
	if errorHandled != nil {
		conditions = append(conditions, taskExModel.ErrorHandled.Is(errorHandled.(bool)))
	}
	_taskExs, err := taskExModel.WithContext(context.Background()).Where(conditions...).Find()
	if err != nil {
		return nil, err
	}

	for _, ex := range _taskExs {
		taskEx := NewTaskExecutionFromDB(ctx, ex)
		taskExs = append(taskExs, taskEx)
	}
	return taskExs, err
}

func QueryCanceledTaskExecutions(ctx *contextx.Context, wfExID string) ([]*TaskExecution, error) {
	var taskExs []*TaskExecution
	var conditions []gen.Condition

	taskExModel := GetQuery(ctx).TaskExecution
	projectId := ctx.GetProjectID()
	conditions = append(conditions, taskExModel.Deleted.Eq(0))
	conditions = append(conditions, taskExModel.WorkflowExecutionID.Eq(wfExID))
	conditions = append(conditions, taskExModel.State.Eq(states.CANCELLED))

	if projectId != "" {
		conditions = append(conditions, taskExModel.ProjectID.Eq(projectId))
	}
	_taskExs, err := taskExModel.WithContext(context.Background()).Where(conditions...).Find()
	if err != nil {
		return nil, err
	}

	for _, ex := range _taskExs {
		taskEx := NewTaskExecutionFromDB(ctx, ex)
		taskExs = append(taskExs, taskEx)
	}
	return taskExs, err
}
