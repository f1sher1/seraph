// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package query

import (
	"context"
	"seraph/app/db/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"gorm.io/gen"
	"gorm.io/gen/field"
)

func newActionDefinition(db *gorm.DB) actionDefinition {
	_actionDefinition := actionDefinition{}

	_actionDefinition.actionDefinitionDo.UseDB(db)
	_actionDefinition.actionDefinitionDo.UseModel(&models.ActionDefinition{})

	tableName := _actionDefinition.actionDefinitionDo.TableName()
	_actionDefinition.ALL = field.NewField(tableName, "*")
	_actionDefinition.ID = field.NewString(tableName, "id")
	_actionDefinition.Name = field.NewString(tableName, "name")
	_actionDefinition.Description = field.NewString(tableName, "description")
	_actionDefinition.Definition = field.NewString(tableName, "definition")
	_actionDefinition.Tags = field.NewField(tableName, "tags")
	_actionDefinition.Spec = field.NewString(tableName, "spec")
	_actionDefinition.Scope = field.NewString(tableName, "scope")
	_actionDefinition.ProjectID = field.NewString(tableName, "project_id")
	_actionDefinition.CreatedAt = field.NewTime(tableName, "created_at")
	_actionDefinition.UpdatedAt = field.NewTime(tableName, "updated_at")
	_actionDefinition.Deleted = field.NewInt(tableName, "deleted")
	_actionDefinition.DeletedAt = field.NewTime(tableName, "deleted_at")
	_actionDefinition.Inputs = field.NewString(tableName, "inputs")
	_actionDefinition.ActionClass = field.NewString(tableName, "action_class")
	_actionDefinition.Attributes = field.NewField(tableName, "attributes")

	_actionDefinition.fillFieldMap()

	return _actionDefinition
}

type actionDefinition struct {
	actionDefinitionDo actionDefinitionDo

	ALL         field.Field
	ID          field.String
	Name        field.String
	Description field.String
	Definition  field.String
	Tags        field.Field
	Spec        field.String
	Scope       field.String
	ProjectID   field.String
	CreatedAt   field.Time
	UpdatedAt   field.Time
	Deleted     field.Int
	DeletedAt   field.Time
	Inputs      field.String
	ActionClass field.String
	Attributes  field.Field

	fieldMap map[string]field.Expr
}

func (a actionDefinition) As(alias string) *actionDefinition {
	a.actionDefinitionDo.DO = *(a.actionDefinitionDo.As(alias).(*gen.DO))

	a.ALL = field.NewField(alias, "*")
	a.ID = field.NewString(alias, "id")
	a.Name = field.NewString(alias, "name")
	a.Description = field.NewString(alias, "description")
	a.Definition = field.NewString(alias, "definition")
	a.Tags = field.NewField(alias, "tags")
	a.Spec = field.NewString(alias, "spec")
	a.Scope = field.NewString(alias, "scope")
	a.ProjectID = field.NewString(alias, "project_id")
	a.CreatedAt = field.NewTime(alias, "created_at")
	a.UpdatedAt = field.NewTime(alias, "updated_at")
	a.Deleted = field.NewInt(alias, "deleted")
	a.DeletedAt = field.NewTime(alias, "deleted_at")
	a.Inputs = field.NewString(alias, "inputs")
	a.ActionClass = field.NewString(alias, "action_class")
	a.Attributes = field.NewField(alias, "attributes")

	a.fillFieldMap()

	return &a
}

func (a *actionDefinition) WithContext(ctx context.Context) *actionDefinitionDo {
	return a.actionDefinitionDo.WithContext(ctx)
}

func (a actionDefinition) TableName() string { return a.actionDefinitionDo.TableName() }

func (a *actionDefinition) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := a.fieldMap[fieldName]
	return _f.(field.OrderExpr), ok
}

func (a *actionDefinition) fillFieldMap() {
	a.fieldMap = make(map[string]field.Expr, 15)
	a.fieldMap["id"] = a.ID
	a.fieldMap["name"] = a.Name
	a.fieldMap["description"] = a.Description
	a.fieldMap["definition"] = a.Definition
	a.fieldMap["tags"] = a.Tags
	a.fieldMap["spec"] = a.Spec
	a.fieldMap["scope"] = a.Scope
	a.fieldMap["project_id"] = a.ProjectID
	a.fieldMap["created_at"] = a.CreatedAt
	a.fieldMap["updated_at"] = a.UpdatedAt
	a.fieldMap["deleted"] = a.Deleted
	a.fieldMap["deleted_at"] = a.DeletedAt
	a.fieldMap["inputs"] = a.Inputs
	a.fieldMap["action_class"] = a.ActionClass
	a.fieldMap["attributes"] = a.Attributes
}

func (a actionDefinition) clone(db *gorm.DB) actionDefinition {
	a.actionDefinitionDo.ReplaceDB(db)
	return a
}

type actionDefinitionDo struct{ gen.DO }

func (a actionDefinitionDo) Debug() *actionDefinitionDo {
	return a.withDO(a.DO.Debug())
}

func (a actionDefinitionDo) WithContext(ctx context.Context) *actionDefinitionDo {
	return a.withDO(a.DO.WithContext(ctx))
}

func (a actionDefinitionDo) Clauses(conds ...clause.Expression) *actionDefinitionDo {
	return a.withDO(a.DO.Clauses(conds...))
}

func (a actionDefinitionDo) Not(conds ...gen.Condition) *actionDefinitionDo {
	return a.withDO(a.DO.Not(conds...))
}

func (a actionDefinitionDo) Or(conds ...gen.Condition) *actionDefinitionDo {
	return a.withDO(a.DO.Or(conds...))
}

func (a actionDefinitionDo) Select(conds ...field.Expr) *actionDefinitionDo {
	return a.withDO(a.DO.Select(conds...))
}

func (a actionDefinitionDo) Where(conds ...gen.Condition) *actionDefinitionDo {
	return a.withDO(a.DO.Where(conds...))
}

func (a actionDefinitionDo) Order(conds ...field.Expr) *actionDefinitionDo {
	return a.withDO(a.DO.Order(conds...))
}

func (a actionDefinitionDo) Distinct(cols ...field.Expr) *actionDefinitionDo {
	return a.withDO(a.DO.Distinct(cols...))
}

func (a actionDefinitionDo) Omit(cols ...field.Expr) *actionDefinitionDo {
	return a.withDO(a.DO.Omit(cols...))
}

func (a actionDefinitionDo) Join(table schema.Tabler, on ...field.Expr) *actionDefinitionDo {
	return a.withDO(a.DO.Join(table, on...))
}

func (a actionDefinitionDo) LeftJoin(table schema.Tabler, on ...field.Expr) *actionDefinitionDo {
	return a.withDO(a.DO.LeftJoin(table, on...))
}

func (a actionDefinitionDo) RightJoin(table schema.Tabler, on ...field.Expr) *actionDefinitionDo {
	return a.withDO(a.DO.RightJoin(table, on...))
}

func (a actionDefinitionDo) Group(cols ...field.Expr) *actionDefinitionDo {
	return a.withDO(a.DO.Group(cols...))
}

func (a actionDefinitionDo) Having(conds ...gen.Condition) *actionDefinitionDo {
	return a.withDO(a.DO.Having(conds...))
}

func (a actionDefinitionDo) Limit(limit int) *actionDefinitionDo {
	return a.withDO(a.DO.Limit(limit))
}

func (a actionDefinitionDo) Offset(offset int) *actionDefinitionDo {
	return a.withDO(a.DO.Offset(offset))
}

func (a actionDefinitionDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *actionDefinitionDo {
	return a.withDO(a.DO.Scopes(funcs...))
}

func (a actionDefinitionDo) Unscoped() *actionDefinitionDo {
	return a.withDO(a.DO.Unscoped())
}

func (a actionDefinitionDo) Create(values ...*models.ActionDefinition) error {
	if len(values) == 0 {
		return nil
	}
	return a.DO.Create(values)
}

func (a actionDefinitionDo) CreateInBatches(values []*models.ActionDefinition, batchSize int) error {
	return a.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (a actionDefinitionDo) Save(values ...*models.ActionDefinition) error {
	if len(values) == 0 {
		return nil
	}
	return a.DO.Save(values)
}

func (a actionDefinitionDo) First() (*models.ActionDefinition, error) {
	if result, err := a.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*models.ActionDefinition), nil
	}
}

func (a actionDefinitionDo) Take() (*models.ActionDefinition, error) {
	if result, err := a.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*models.ActionDefinition), nil
	}
}

func (a actionDefinitionDo) Last() (*models.ActionDefinition, error) {
	if result, err := a.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*models.ActionDefinition), nil
	}
}

func (a actionDefinitionDo) Find() ([]*models.ActionDefinition, error) {
	result, err := a.DO.Find()
	return result.([]*models.ActionDefinition), err
}

func (a actionDefinitionDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*models.ActionDefinition, err error) {
	buf := make([]*models.ActionDefinition, 0, batchSize)
	err = a.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (a actionDefinitionDo) FindInBatches(result *[]*models.ActionDefinition, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return a.DO.FindInBatches(result, batchSize, fc)
}

func (a actionDefinitionDo) Attrs(attrs ...field.AssignExpr) *actionDefinitionDo {
	return a.withDO(a.DO.Attrs(attrs...))
}

func (a actionDefinitionDo) Assign(attrs ...field.AssignExpr) *actionDefinitionDo {
	return a.withDO(a.DO.Assign(attrs...))
}

func (a actionDefinitionDo) Joins(field field.RelationField) *actionDefinitionDo {
	return a.withDO(a.DO.Joins(field))
}

func (a actionDefinitionDo) Preload(field field.RelationField) *actionDefinitionDo {
	return a.withDO(a.DO.Preload(field))
}

func (a actionDefinitionDo) FirstOrInit() (*models.ActionDefinition, error) {
	if result, err := a.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*models.ActionDefinition), nil
	}
}

func (a actionDefinitionDo) FirstOrCreate() (*models.ActionDefinition, error) {
	if result, err := a.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*models.ActionDefinition), nil
	}
}

func (a actionDefinitionDo) FindByPage(offset int, limit int) (result []*models.ActionDefinition, count int64, err error) {
	count, err = a.Count()
	if err != nil {
		return
	}

	result, err = a.Offset(offset).Limit(limit).Find()
	return
}

func (a actionDefinitionDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = a.Count()
	if err != nil {
		return
	}

	err = a.Offset(offset).Limit(limit).Scan(result)
	return
}

func (a *actionDefinitionDo) withDO(do gen.Dao) *actionDefinitionDo {
	a.DO = *do.(*gen.DO)
	return a
}
