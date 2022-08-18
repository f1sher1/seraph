package objects

import (
	"context"
	"encoding/json"
	"fmt"
	"seraph/app/db/models"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ActionDefinition struct {
	*models.ActionDefinition
	ContextObject
	PersistentObject
}

func (d ActionDefinition) GetMergedInput(input map[string]interface{}) map[string]interface{} {
	mergedInput := map[string]interface{}{}
	for k, v := range input {
		mergedInput[k] = v
	}

	if d.Inputs != "" {
		inputDef := map[string]interface{}{}

		paramStrSlice := strings.Split(d.Inputs, ",")
		for _, paramStr := range paramStrSlice {
			paramStr = strings.TrimSpace(paramStr)
			key := paramStr
			eqPos := strings.Index(paramStr, "=")
			if eqPos > -1 {
				key = paramStr[:eqPos]
				value := paramStr[eqPos+1:]
				if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
					inputDef[key] = value[1 : len(value)-1]
				} else {
					var v interface{}
					if err := json.Unmarshal([]byte(value), &v); err == nil {
						inputDef[key] = v
					} else {
						inputDef[key] = value
					}
				}
			} else {
				inputDef[key] = nil
			}
		}

		for k, v := range inputDef {
			if _, ok := mergedInput[k]; !ok {
				mergedInput[k] = v
			}
		}
	}
	return mergedInput
}

func (d ActionDefinition) GetSpec() *ActionSpec {
	if d.Spec == "" {
		return nil
	}
	spec := &ActionSpec{}
	err := json.Unmarshal([]byte(d.Spec), spec)
	if err != nil {
		log.Errorf(nil, "parse action spec failed, error: %s", err.Error())
	}
	err = spec.Initialize()
	if err != nil {
		log.Errorf(nil, "initial action spec failed, error: %s", err.Error())
	}
	return spec
}

func (d *ActionDefinition) Save(ctx *contextx.Context) error {
	if !d.IsCreated() {
		d.CreatedAt = time.Now().UTC()
		if d.ID == "" {
			d.ID = uuid.NewString()
		}
		d.UpdatedAt = d.CreatedAt
	} else {
		d.UpdatedAt = time.Now().UTC()
	}

	dbModel := d.GetQuery(ctx).ActionDefinition
	err := dbModel.WithContext(context.Background()).Save(d.ActionDefinition)
	if err != nil {
		return err
	}
	d.SetContext(ctx)
	d.SetCreated()
	return nil
}

func (d *ActionDefinition) Delete(ctx *contextx.Context) error {
	if !d.IsCreated() {
		return fmt.Errorf("object %s isn't a persistent object, can't delete it", d.ID)
	}

	d.Deleted = 1
	d.DeletedAt = time.Now().UTC()
	return d.Save(ctx)
}

func NewActionDefinition() *ActionDefinition {
	return &ActionDefinition{
		ActionDefinition: &models.ActionDefinition{},
	}
}

func NewActionDefinitionFromDB(ctx *contextx.Context, definition *models.ActionDefinition) *ActionDefinition {
	if definition == nil {
		return nil
	}
	def := &ActionDefinition{
		ActionDefinition: definition,
	}
	def.SetContext(ctx)
	def.SetCreated()
	return def
}

func QueryActionDefinitionByID(ctx *contextx.Context, id string) (*ActionDefinition, error) {
	actionDefModel := GetQuery(ctx).ActionDefinition

	_actionDef, err := actionDefModel.WithContext(context.Background()).Where(
		actionDefModel.ID.Eq(id),
		actionDefModel.Deleted.Eq(0),
	).First()

	if IsNotFoundError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return NewActionDefinitionFromDB(ctx, _actionDef), nil
}

func QueryActionDefinitionByName(ctx *contextx.Context, name string) (*ActionDefinition, error) {
	actionDefModel := GetQuery(ctx).ActionDefinition
	q := actionDefModel.WithContext(context.Background()).Where(
		actionDefModel.Name.Eq(name),
		actionDefModel.Deleted.Eq(0),
	)

	if !ctx.IsAdmin() {
		q = q.Where(actionDefModel.WithContext(context.Background()).Where(
			actionDefModel.Scope.Eq("public"),
		).Or(
			actionDefModel.Scope.Eq("private"),
			actionDefModel.ProjectID.Eq(ctx.GetProjectID()),
		))
	}

	_actionDef, err := q.First()
	if IsNotFoundError(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return NewActionDefinitionFromDB(ctx, _actionDef), nil
}
