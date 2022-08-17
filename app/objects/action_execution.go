package objects

import (
	"context"
	"fmt"
	"seraph/app/db/models"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gen/field"
)

type ActionExecution struct {
	*models.ActionExecution
	ContextObject
	PersistentObject
}

func (e *ActionExecution) Save(ctx *contextx.Context) error {
	if !e.isCreated {
		e.CreatedAt = time.Now().UTC()
		if e.ID == "" {
			e.ID = uuid.NewString()
		}
		e.UpdatedAt = e.CreatedAt
	} else {
		e.UpdatedAt = time.Now().UTC()
	}

	actionExModel := e.GetQuery(ctx).ActionExecution
	err := actionExModel.WithContext(context.Background()).Save(e.ActionExecution)
	if err != nil {
		return nil
	}
	e.SetContext(ctx)
	e.SetCreated()
	return nil
}

func (e *ActionExecution) Update(ctx *contextx.Context, fields ...string) error {
	actionExModel := e.GetQuery(ctx).ActionExecution
	var flds []field.Expr
	for _, f := range fields {
		switch f {
		case "State":
			flds = append(flds, actionExModel.State)
		case "Outputs":
			flds = append(flds, actionExModel.Outputs)
		case "FinishedAt":
			flds = append(flds, actionExModel.FinishedAt)
		case "Deleted":
			flds = append(flds, actionExModel.Deleted)
		case "DeletedAt":
			flds = append(flds, actionExModel.DeletedAt)
		}

	}
	result, err := actionExModel.WithContext(context.Background()).Select(flds...).Where(actionExModel.ID.Eq(e.ID)).Updates(e.ActionExecution)
	if err != nil {
		log.Errorf(ctx, "Save data error: %v", err.Error())
		return err
	}
	log.Debugf(ctx, "Update data result %#v", result)
	return nil
}

func (e *ActionExecution) Delete(ctx *contextx.Context) error {
	if !e.IsCreated() {
		return fmt.Errorf("object %s isn't a persistent object, can't delete it", e.ID)
	}
	e.Deleted = 1
	e.DeletedAt = time.Now().UTC()
	// return e.Save(ctx)
	return e.Update(ctx, "Deleted", "DeletedAt")
}

func (e *ActionExecution) GetTaskExecution() (*TaskExecution, error) {
	return QueryTaskExecutionByID(e.GetContext(), e.TaskExecutionID)
}

func NewActionExecution() *ActionExecution {
	return &ActionExecution{
		ActionExecution: &models.ActionExecution{},
	}
}

func NewActionExecutionFromDB(ctx *contextx.Context, ex *models.ActionExecution) *ActionExecution {
	if ex == nil {
		return nil
	}
	def := &ActionExecution{
		ActionExecution: ex,
	}
	def.SetContext(ctx)
	def.SetCreated()
	return def
}

func QueryActionExecutionsByTaskID(ctx *contextx.Context, id string) ([]*ActionExecution, error) {
	actionExModel := GetQuery(ctx).ActionExecution

	_actionExs, err := actionExModel.WithContext(context.Background()).Where(
		actionExModel.TaskExecutionID.Eq(id),
		actionExModel.Deleted.Eq(0),
	).Find()

	if err != nil {
		return nil, err
	}

	var actionExs []*ActionExecution
	for _, _ex := range _actionExs {
		ex := NewActionExecutionFromDB(ctx, _ex)
		actionExs = append(actionExs, ex)
	}
	return actionExs, nil
}

func QueryActionExecutionByID(ctx *contextx.Context, id string) (*ActionExecution, error) {
	actionExModel := GetQuery(ctx).ActionExecution

	_actionEx, err := actionExModel.WithContext(context.Background()).Where(
		actionExModel.ID.Eq(id),
		actionExModel.Deleted.Eq(0),
	).First()

	if IsNotFoundError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return NewActionExecutionFromDB(ctx, _actionEx), nil
}
