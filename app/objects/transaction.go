package objects

import (
	"seraph/app/db/query"
	"seraph/pkg/contextx"
)

func Transaction(ctx *contextx.Context, fc func(subCtx *contextx.Context) error) error {
	subCtx := ctx.Clone()
	return GetQuery(ctx).Transaction(func(tx *query.Query) error {
		subCtx.SetDB(tx)
		return fc(subCtx)
	})
}
