package contextx

import "context"

const (
	AdminRoleName = "admin"
)
type Role map[string]interface{}
type Roles []Role

func (r Role) Name() string {
	if name, ok := r["name"]; ok {
		return name.(string)
	}
	return ""
}

type Context struct {
	context.Context
	dbTx interface{}
	data map[string]interface{}
	adminRole bool
}

func (ctx *Context) Clone() *Context {
	newCtx :=  &Context{
		Context:   context.Background(),
		data:      map[string]interface{} {},
		adminRole: ctx.adminRole,
	}
	for k, v := range ctx.data {
		newCtx.data[k] = v
	}

	return newCtx
}

func (ctx *Context) Set(name string, value interface{}) {
	ctx.data[name] = value
}

func (ctx *Context) GetDB() interface{} {
	return ctx.dbTx
}

func (ctx *Context) SetDB(tx interface{}) {
	ctx.dbTx = tx
}

func (ctx *Context) GetMap() map[string]interface{} {
	return ctx.data
}

func (ctx *Context) GetProjectID() string {
	if projectId, ok := ctx.data["project_id"]; ok {
		return projectId.(string)
	}
	return ""
}

func (ctx *Context) IsAdmin() bool {
	if ctx.adminRole {
		return true
	}
	if roles, ok := ctx.data["roles"]; ok {
		for _, role := range roles.(Roles) {
			if role.Name() == AdminRoleName {
				return true
			}
		}
	}
	return false
}

func NewContext() *Context {
	return &Context{
		Context: context.Background(),
		data:    map[string]interface{}{},
	}
}

func NewAdminContext() *Context {
	return &Context{
		Context: context.Background(),
		data:    map[string]interface{}{},
		adminRole: true,
	}
}

func NewContextFromMap(data map[string]interface{}) *Context {
	return &Context{
		Context: context.Background(),
		data:    data,
	}
}
