package objects

import (
	"context"
	"encoding/json"
	"fmt"
	"seraph/app/db"
	"seraph/app/db/models"
	"seraph/app/db/query"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

type WorkflowExecution struct {
	*models.WorkflowExecution
	ContextObject
	PersistentObject
}

func (e *WorkflowExecution) GetSpec() (*WorkflowSpec, error) {
	spec := &WorkflowSpec{}
	if err := json.Unmarshal([]byte(e.Spec), &spec); err != nil {
		return nil, err
	}
	if err := spec.Initialize(); err != nil {
		return nil, err
	}
	return spec, nil
}

func (e *WorkflowExecution) GetChildrenTaskExecutions() ([]*TaskExecution, error) {
	return QueryTaskExecutions(e.GetContext(), e.ID, nil, nil, nil, nil)
}

func (e *WorkflowExecution) GetTaskExecution() (*TaskExecution, error) {
	return QueryTaskExecutionByID(e.GetContext(), e.TaskExecutionID)
}

func (e *WorkflowExecution) Save(ctx *contextx.Context) error {
	if !e.IsCreated() {
		e.CreatedAt = time.Now().UTC()
		if e.ID == "" {
			e.ID = uuid.NewString()
		}
		e.UpdatedAt = e.CreatedAt
	} else {
		e.UpdatedAt = time.Now().UTC()
	}

	exModel := e.GetQuery(ctx).WorkflowExecution
	err := exModel.WithContext(context.Background()).Save(e.WorkflowExecution)
	if err != nil {
		return err
	}
	e.SetContext(ctx)
	e.SetCreated()
	return nil
}
func (e *WorkflowExecution) Update(ctx *contextx.Context, fields ...string) error {
	exModel := e.GetQuery(ctx).WorkflowExecution
	var flds []field.Expr
	for _, f := range fields {
		switch f {
		case "State":
			flds = append(flds, exModel.State)
		case "StateInfo":
			flds = append(flds, exModel.StateInfo)
		case "Output":
			flds = append(flds, exModel.Output)
		case "FinishedAt":
			flds = append(flds, exModel.FinishedAt)
		case "Deleted":
			flds = append(flds, exModel.Deleted)
		case "DeletedAt":
			flds = append(flds, exModel.DeletedAt)
		}
	}
	result, err := exModel.WithContext(context.Background()).Select(flds...).Where(exModel.ID.Eq(e.ID)).Updates(e.WorkflowExecution)
	if err != nil {
		log.Errorf(ctx, "Save data error: %v", err.Error())
		return err
	}
	log.Debugf(ctx, "Update data result %#v", result)
	return nil
}

func (e *WorkflowExecution) Delete(ctx *contextx.Context) error {
	if !e.IsCreated() {
		return fmt.Errorf("object %s isn't a persistent object, can't delete it", e.ID)
	}

	e.Deleted = 1
	e.DeletedAt = time.Now().UTC()
	// return e.Save(ctx)
	return e.Update(ctx, "Deleted", "DeletedAt")
}

func NewWorkflowExecution() *WorkflowExecution {
	return &WorkflowExecution{
		WorkflowExecution: &models.WorkflowExecution{},
	}
}

func NewWorkflowExecutionFromDB(ctx *contextx.Context, ex *models.WorkflowExecution) *WorkflowExecution {
	if ex == nil {
		return nil
	}
	wfEx := &WorkflowExecution{
		WorkflowExecution: ex,
	}
	wfEx.SetContext(ctx)
	wfEx.SetCreated()
	return wfEx
}

func QueryWorkflowExecutions(ctx *contextx.Context, name, namespace interface{}) ([]*WorkflowExecution, error) {
	var wfExs []*WorkflowExecution
	var conditions []gen.Condition

	exModel := GetQuery(ctx).WorkflowExecution
	projectId := ctx.GetProjectID()
	conditions = append(conditions, exModel.Deleted.Eq(0))

	if projectId != "" {
		conditions = append(conditions, exModel.ProjectID.Eq(projectId))
	}
	if name != nil {
		conditions = append(conditions, exModel.Name.Eq(name.(string)))
	}
	if namespace != nil {
		conditions = append(conditions, exModel.WorkflowNamespace.Eq(namespace.(string)))
	}

	_wfExs, err := exModel.WithContext(context.Background()).Where(conditions...).Find()
	if err != nil {
		return nil, err
	}

	for _, ex := range _wfExs {
		wfEx := NewWorkflowExecutionFromDB(ctx, ex)
		wfExs = append(wfExs, wfEx)
	}
	return wfExs, err
}

func QueryWorkflowExecutionByID(ctx *contextx.Context, id string) (*WorkflowExecution, error) {
	var conditions []gen.Condition

	exModel := query.Use(db.GetDBConnection()).WorkflowExecution
	projectId := ctx.GetProjectID()
	conditions = append(conditions, exModel.Deleted.Eq(0))
	conditions = append(conditions, exModel.ID.Eq(id))

	if projectId != "" {
		conditions = append(conditions, exModel.ProjectID.Eq(projectId))
	}

	_wfEx, err := exModel.WithContext(context.Background()).Where(conditions...).First()
	if IsNotFoundError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if _wfEx == nil {
		return nil, nil
	}

	wfEx := NewWorkflowExecutionFromDB(ctx, _wfEx)
	return wfEx, nil
}

func QueryWorkflowExecutionByTaskExecutionID(ctx *contextx.Context, id string) (*WorkflowExecution, error) {
	var conditions []gen.Condition

	exModel := query.Use(db.GetDBConnection()).WorkflowExecution
	projectId := ctx.GetProjectID()
	conditions = append(conditions, exModel.Deleted.Eq(0))
	conditions = append(conditions, exModel.TaskExecutionID.Eq(id))

	if projectId != "" {
		conditions = append(conditions, exModel.ProjectID.Eq(projectId))
	}

	_wfEx, err := exModel.WithContext(context.Background()).Where(conditions...).First()
	if IsNotFoundError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if _wfEx == nil {
		return nil, nil
	}

	wfEx := NewWorkflowExecutionFromDB(ctx, _wfEx)
	return wfEx, nil
}
