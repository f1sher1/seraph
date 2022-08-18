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

func newNamedLock(db *gorm.DB) namedLock {
	_namedLock := namedLock{}

	_namedLock.namedLockDo.UseDB(db)
	_namedLock.namedLockDo.UseModel(&models.NamedLock{})

	tableName := _namedLock.namedLockDo.TableName()
	_namedLock.ALL = field.NewField(tableName, "*")
	_namedLock.ID = field.NewString(tableName, "id")
	_namedLock.Name = field.NewString(tableName, "name")
	_namedLock.CreatedAt = field.NewTime(tableName, "created_at")
	_namedLock.UpdatedAt = field.NewTime(tableName, "updated_at")

	_namedLock.fillFieldMap()

	return _namedLock
}

type namedLock struct {
	namedLockDo namedLockDo

	ALL       field.Field
	ID        field.String
	Name      field.String
	CreatedAt field.Time
	UpdatedAt field.Time

	fieldMap map[string]field.Expr
}

func (n namedLock) As(alias string) *namedLock {
	n.namedLockDo.DO = *(n.namedLockDo.As(alias).(*gen.DO))

	n.ALL = field.NewField(alias, "*")
	n.ID = field.NewString(alias, "id")
	n.Name = field.NewString(alias, "name")
	n.CreatedAt = field.NewTime(alias, "created_at")
	n.UpdatedAt = field.NewTime(alias, "updated_at")

	n.fillFieldMap()

	return &n
}

func (n *namedLock) WithContext(ctx context.Context) *namedLockDo {
	return n.namedLockDo.WithContext(ctx)
}

func (n namedLock) TableName() string { return n.namedLockDo.TableName() }

func (n *namedLock) GetFieldByName(fieldName string) (field.OrderExpr, bool) {
	_f, ok := n.fieldMap[fieldName]
	return _f.(field.OrderExpr), ok
}

func (n *namedLock) fillFieldMap() {
	n.fieldMap = make(map[string]field.Expr, 4)
	n.fieldMap["id"] = n.ID
	n.fieldMap["name"] = n.Name
	n.fieldMap["created_at"] = n.CreatedAt
	n.fieldMap["updated_at"] = n.UpdatedAt
}

func (n namedLock) clone(db *gorm.DB) namedLock {
	n.namedLockDo.ReplaceDB(db)
	return n
}

type namedLockDo struct{ gen.DO }

func (n namedLockDo) Debug() *namedLockDo {
	return n.withDO(n.DO.Debug())
}

func (n namedLockDo) WithContext(ctx context.Context) *namedLockDo {
	return n.withDO(n.DO.WithContext(ctx))
}

func (n namedLockDo) Clauses(conds ...clause.Expression) *namedLockDo {
	return n.withDO(n.DO.Clauses(conds...))
}

func (n namedLockDo) Not(conds ...gen.Condition) *namedLockDo {
	return n.withDO(n.DO.Not(conds...))
}

func (n namedLockDo) Or(conds ...gen.Condition) *namedLockDo {
	return n.withDO(n.DO.Or(conds...))
}

func (n namedLockDo) Select(conds ...field.Expr) *namedLockDo {
	return n.withDO(n.DO.Select(conds...))
}

func (n namedLockDo) Where(conds ...gen.Condition) *namedLockDo {
	return n.withDO(n.DO.Where(conds...))
}

func (n namedLockDo) Order(conds ...field.Expr) *namedLockDo {
	return n.withDO(n.DO.Order(conds...))
}

func (n namedLockDo) Distinct(cols ...field.Expr) *namedLockDo {
	return n.withDO(n.DO.Distinct(cols...))
}

func (n namedLockDo) Omit(cols ...field.Expr) *namedLockDo {
	return n.withDO(n.DO.Omit(cols...))
}

func (n namedLockDo) Join(table schema.Tabler, on ...field.Expr) *namedLockDo {
	return n.withDO(n.DO.Join(table, on...))
}

func (n namedLockDo) LeftJoin(table schema.Tabler, on ...field.Expr) *namedLockDo {
	return n.withDO(n.DO.LeftJoin(table, on...))
}

func (n namedLockDo) RightJoin(table schema.Tabler, on ...field.Expr) *namedLockDo {
	return n.withDO(n.DO.RightJoin(table, on...))
}

func (n namedLockDo) Group(cols ...field.Expr) *namedLockDo {
	return n.withDO(n.DO.Group(cols...))
}

func (n namedLockDo) Having(conds ...gen.Condition) *namedLockDo {
	return n.withDO(n.DO.Having(conds...))
}

func (n namedLockDo) Limit(limit int) *namedLockDo {
	return n.withDO(n.DO.Limit(limit))
}

func (n namedLockDo) Offset(offset int) *namedLockDo {
	return n.withDO(n.DO.Offset(offset))
}

func (n namedLockDo) Scopes(funcs ...func(gen.Dao) gen.Dao) *namedLockDo {
	return n.withDO(n.DO.Scopes(funcs...))
}

func (n namedLockDo) Unscoped() *namedLockDo {
	return n.withDO(n.DO.Unscoped())
}

func (n namedLockDo) Create(values ...*models.NamedLock) error {
	if len(values) == 0 {
		return nil
	}
	return n.DO.Create(values)
}

func (n namedLockDo) CreateInBatches(values []*models.NamedLock, batchSize int) error {
	return n.DO.CreateInBatches(values, batchSize)
}

// Save : !!! underlying implementation is different with GORM
// The method is equivalent to executing the statement: db.Clauses(clause.OnConflict{UpdateAll: true}).Create(values)
func (n namedLockDo) Save(values ...*models.NamedLock) error {
	if len(values) == 0 {
		return nil
	}
	return n.DO.Save(values)
}

func (n namedLockDo) First() (*models.NamedLock, error) {
	if result, err := n.DO.First(); err != nil {
		return nil, err
	} else {
		return result.(*models.NamedLock), nil
	}
}

func (n namedLockDo) Take() (*models.NamedLock, error) {
	if result, err := n.DO.Take(); err != nil {
		return nil, err
	} else {
		return result.(*models.NamedLock), nil
	}
}

func (n namedLockDo) Last() (*models.NamedLock, error) {
	if result, err := n.DO.Last(); err != nil {
		return nil, err
	} else {
		return result.(*models.NamedLock), nil
	}
}

func (n namedLockDo) Find() ([]*models.NamedLock, error) {
	result, err := n.DO.Find()
	return result.([]*models.NamedLock), err
}

func (n namedLockDo) FindInBatch(batchSize int, fc func(tx gen.Dao, batch int) error) (results []*models.NamedLock, err error) {
	buf := make([]*models.NamedLock, 0, batchSize)
	err = n.DO.FindInBatches(&buf, batchSize, func(tx gen.Dao, batch int) error {
		defer func() { results = append(results, buf...) }()
		return fc(tx, batch)
	})
	return results, err
}

func (n namedLockDo) FindInBatches(result *[]*models.NamedLock, batchSize int, fc func(tx gen.Dao, batch int) error) error {
	return n.DO.FindInBatches(result, batchSize, fc)
}

func (n namedLockDo) Attrs(attrs ...field.AssignExpr) *namedLockDo {
	return n.withDO(n.DO.Attrs(attrs...))
}

func (n namedLockDo) Assign(attrs ...field.AssignExpr) *namedLockDo {
	return n.withDO(n.DO.Assign(attrs...))
}

func (n namedLockDo) Joins(field field.RelationField) *namedLockDo {
	return n.withDO(n.DO.Joins(field))
}

func (n namedLockDo) Preload(field field.RelationField) *namedLockDo {
	return n.withDO(n.DO.Preload(field))
}

func (n namedLockDo) FirstOrInit() (*models.NamedLock, error) {
	if result, err := n.DO.FirstOrInit(); err != nil {
		return nil, err
	} else {
		return result.(*models.NamedLock), nil
	}
}

func (n namedLockDo) FirstOrCreate() (*models.NamedLock, error) {
	if result, err := n.DO.FirstOrCreate(); err != nil {
		return nil, err
	} else {
		return result.(*models.NamedLock), nil
	}
}

func (n namedLockDo) FindByPage(offset int, limit int) (result []*models.NamedLock, count int64, err error) {
	count, err = n.Count()
	if err != nil {
		return
	}

	result, err = n.Offset(offset).Limit(limit).Find()
	return
}

func (n namedLockDo) ScanByPage(result interface{}, offset int, limit int) (count int64, err error) {
	count, err = n.Count()
	if err != nil {
		return
	}

	err = n.Offset(offset).Limit(limit).Scan(result)
	return
}

func (n *namedLockDo) withDO(do gen.Dao) *namedLockDo {
	n.DO = *do.(*gen.DO)
	return n
}
