package objects

import (
	"context"
	"encoding/json"
	"fmt"
	"seraph/app/db/models"
	"seraph/pkg/contextx"
	"time"

	"github.com/google/uuid"
	"gorm.io/gen"
)

type WorkflowDefinition struct {
	*models.WorkflowDefinition
	ContextObject
	PersistentObject
}

func (d WorkflowDefinition) GetSpec() (*WorkflowSpec, error) {
	wfSpec := &WorkflowSpec{}
	if err := json.Unmarshal([]byte(d.Spec), &wfSpec); err != nil {
		return nil, err
	}
	if err := wfSpec.Initialize(); err != nil {
		return nil, err
	}
	return wfSpec, nil
}

func (d *WorkflowDefinition) Save(ctx *contextx.Context) error {
	if !d.IsCreated() {
		d.CreatedAt = time.Now().UTC()
		if d.ID == "" {
			d.ID = uuid.NewString()
		}
		d.UpdatedAt = d.CreatedAt
	} else {
		d.UpdatedAt = time.Now().UTC()
	}

	dbModel := d.GetQuery(ctx).WorkflowDefinition
	err := dbModel.WithContext(context.Background()).Save(d.WorkflowDefinition)
	if err != nil {
		return err
	}
	d.SetContext(ctx)
	d.SetCreated()
	return nil
}

func (d *WorkflowDefinition) Delete(ctx *contextx.Context) error {
	if !d.IsCreated() {
		return fmt.Errorf("object %s isn't a persistent object, can't delete it", d.ID)
	}

	d.Deleted = 1
	d.DeletedAt = time.Now().UTC()
	return d.Save(ctx)
}

func NewWorkflowDefinition() *WorkflowDefinition {
	return &WorkflowDefinition{
		WorkflowDefinition: &models.WorkflowDefinition{},
	}
}

func NewWorkflowDefinitionFromDB(ctx *contextx.Context, def *models.WorkflowDefinition) *WorkflowDefinition {
	if def == nil {
		return nil
	}
	wfDef := &WorkflowDefinition{
		WorkflowDefinition: def,
	}
	wfDef.SetContext(ctx)
	wfDef.SetCreated()
	return wfDef
}

func QueryWorkflowDefinitions(ctx *contextx.Context, name, namespace interface{}) ([]*WorkflowDefinition, error) {
	var wfDefinitions []*WorkflowDefinition
	var conditions []gen.Condition

	defModel := GetQuery(ctx).WorkflowDefinition
	conditions = append(conditions, defModel.Deleted.Eq(0))

	if ctx.GetProjectID() != "" {
		conditions = append(conditions, defModel.ProjectID.Eq(ctx.GetProjectID()))
	}
	if name != nil {
		conditions = append(conditions, defModel.Name.Eq(name.(string)))
	}
	if namespace != nil {
		conditions = append(conditions, defModel.Namespace.Eq(namespace.(string)))
	}

	_wfDefinitions, err := defModel.WithContext(context.Background()).Where(conditions...).Find()
	if err != nil {
		return nil, err
	}

	for _, def := range _wfDefinitions {
		wfDef := NewWorkflowDefinitionFromDB(ctx, def)
		wfDefinitions = append(wfDefinitions, wfDef)
	}
	return wfDefinitions, err
}

func QueryWorkflowDefinitionByID(ctx *contextx.Context, id string) (*WorkflowDefinition, error) {
	var conditions []gen.Condition

	defModel := GetQuery(ctx).WorkflowDefinition
	conditions = append(conditions, defModel.Deleted.Eq(0))
	conditions = append(conditions, defModel.ID.Eq(id))

	if ctx.GetProjectID() != "" {
		conditions = append(conditions, defModel.ProjectID.Eq(ctx.GetProjectID()))
	}

	_wfDef, err := defModel.WithContext(context.Background()).Where(conditions...).First()
	if IsNotFoundError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if _wfDef == nil {
		return nil, fmt.Errorf("workflow definition with id '%s' not found", id)
	}
	return NewWorkflowDefinitionFromDB(ctx, _wfDef), nil
}
