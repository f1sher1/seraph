package objects

import (
	"encoding/json"
	"seraph/app/db"
	"seraph/app/db/query"
	"seraph/pkg/contextx"
	"seraph/pkg/log"

	"gorm.io/gorm"
)

type Table map[string]interface{}

func (t Table) Has(name string) bool {
	_, ok := t[name]
	return ok
}

func (t Table) Get(name string) interface{} {
	v, ok := t[name]
	if ok {
		return v
	}
	return nil
}

func (t Table) ToString() string {
	str, err := json.Marshal(t)
	if err != nil {
		log.Errorf(nil, "json marshal failed, error: %s", err.Error())
		return ""
	}
	return string(str)
}

func (t Table) MergeToNew(tabs ...Table) Table {
	newT := Table{}
	for k, v := range t {
		newT[k] = v
	}

	for _, tab := range tabs {
		for k, v := range tab {
			newT[k] = v
		}
	}
	return newT
}

func NewTableFromString(data string) (*Table, error) {
	t := &Table{}
	err := json.Unmarshal([]byte(data), t)
	return t, err
}

type SliceString []string

func (s SliceString) Has(value string) bool {
	for _, v := range s {
		if v == value {
			return true
		}
	}
	return false
}

type SliceInterface []interface{}

func (s SliceInterface) Has(value interface{}) bool {
	for _, v := range s {
		if v == value {
			return true
		}
	}
	return false
}

func GetQuery(ctx *contextx.Context) *query.Query {
	var dbTx *gorm.DB
	if ctx == nil || ctx.GetDB() == nil {
		dbTx = db.GetDBConnection()
		return query.Use(dbTx)
	} else {
		return ctx.GetDB().(*query.Query)
	}
}

type ContextObject struct {
	ctx *contextx.Context
}

func (c *ContextObject) GetContext() *contextx.Context {
	return c.ctx
}

func (c *ContextObject) SetContext(ctx *contextx.Context) {
	if ctx != nil {
		c.ctx = ctx
	}
}

func (c *ContextObject) GetQuery(ctx *contextx.Context) *query.Query {
	if ctx == nil {
		ctx = c.GetContext()
	}
	return GetQuery(ctx)
}

type PersistentObject struct {
	isCreated bool
}

func (p *PersistentObject) IsCreated() bool {
	return p.isCreated
}

func (p *PersistentObject) SetCreated() {
	if !p.isCreated {
		p.isCreated = true
	}
}
