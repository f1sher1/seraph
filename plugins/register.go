package plugins

import (
	"seraph/app/db/query"
	"seraph/app/objects"
	"seraph/pkg/contextx"
	"seraph/plugins/nova"
	"seraph/plugins/standard"
)

func RegisterBuiltinActions() error {
	ctx := contextx.NewAdminContext()
	q := objects.GetQuery(ctx)

	return q.Transaction(func(tx *query.Query) error {
		ctx.SetDB(tx)

		endpoints := nova.GetEndpoints()
		endpoints_std := standard.GetEndpoints()
		for k, v := range endpoints_std {
			endpoints[k] = v
		}
		for name := range endpoints {
			actDef, err := objects.QueryActionDefinitionByName(ctx, name)
			if err != nil {
				return err
			}

			if actDef == nil {
				actDef = objects.NewActionDefinition()
			}

			actDef.Scope = "public"
			actDef.ActionClass = name
			actDef.Name = name
			err = actDef.Save(ctx)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
