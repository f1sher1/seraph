// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package query

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

func Use(db *gorm.DB) *Query {
	return &Query{
		db:                 db,
		ActionDefinition:   newActionDefinition(db),
		ActionExecution:    newActionExecution(db),
		NamedLock:          newNamedLock(db),
		TaskExecution:      newTaskExecution(db),
		WorkflowDefinition: newWorkflowDefinition(db),
		WorkflowExecution:  newWorkflowExecution(db),
	}
}

type Query struct {
	db *gorm.DB

	ActionDefinition   actionDefinition
	ActionExecution    actionExecution
	NamedLock          namedLock
	TaskExecution      taskExecution
	WorkflowDefinition workflowDefinition
	WorkflowExecution  workflowExecution
}

func (q *Query) Available() bool { return q.db != nil }

func (q *Query) clone(db *gorm.DB) *Query {
	return &Query{
		db:                 db,
		ActionDefinition:   q.ActionDefinition.clone(db),
		ActionExecution:    q.ActionExecution.clone(db),
		NamedLock:          q.NamedLock.clone(db),
		TaskExecution:      q.TaskExecution.clone(db),
		WorkflowDefinition: q.WorkflowDefinition.clone(db),
		WorkflowExecution:  q.WorkflowExecution.clone(db),
	}
}

type queryCtx struct {
	ActionDefinition   actionDefinitionDo
	ActionExecution    actionExecutionDo
	NamedLock          namedLockDo
	TaskExecution      taskExecutionDo
	WorkflowDefinition workflowDefinitionDo
	WorkflowExecution  workflowExecutionDo
}

func (q *Query) WithContext(ctx context.Context) *queryCtx {
	return &queryCtx{
		ActionDefinition:   *q.ActionDefinition.WithContext(ctx),
		ActionExecution:    *q.ActionExecution.WithContext(ctx),
		NamedLock:          *q.NamedLock.WithContext(ctx),
		TaskExecution:      *q.TaskExecution.WithContext(ctx),
		WorkflowDefinition: *q.WorkflowDefinition.WithContext(ctx),
		WorkflowExecution:  *q.WorkflowExecution.WithContext(ctx),
	}
}

func (q *Query) Transaction(fc func(tx *Query) error, opts ...*sql.TxOptions) error {
	return q.db.Transaction(func(tx *gorm.DB) error { return fc(q.clone(tx)) }, opts...)
}

func (q *Query) Begin(opts ...*sql.TxOptions) *QueryTx {
	return &QueryTx{q.clone(q.db.Begin(opts...))}
}

type QueryTx struct{ *Query }

func (q *QueryTx) Commit() error {
	return q.db.Commit().Error
}

func (q *QueryTx) Rollback() error {
	return q.db.Rollback().Error
}

func (q *QueryTx) SavePoint(name string) error {
	return q.db.SavePoint(name).Error
}

func (q *QueryTx) RollbackTo(name string) error {
	return q.db.RollbackTo(name).Error
}
