package objects

import (
	"seraph/app/db"
	"seraph/pkg/contextx"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestActionDefinition_Create(t *testing.T) {
	asserter := assert.New(t)

	cfg := &db.Config{
		Connection:  "sqlite:///tmp/test.db",
		Debug:       true,
		PoolSize:    5,
		IdleTimeout: 3600,
	}
	err := db.Init(cfg)

	if asserter.NoError(err) {
		err = db.Migrate()

		if asserter.NoError(err) {
			ctx := contextx.NewContext()
			ctx.Set("project_id", "project_id_1")

			def := NewActionDefinition()
			def.ID = uuid.NewString()
			def.Name = "std.echo"
			def.ActionClass = "std.echo-test"
			def.Attributes = map[string]interface{}{
				"attr1": "attr1value",
			}
			def.Inputs = `{"a": 1}`
			def.Description = "this is echo action"
			err = def.Save(ctx)

			if asserter.NoError(err) {
				def2, err := QueryActionDefinitionByID(ctx, def.ID)
				if asserter.NoError(err) {
					asserter.Equal(*def.ActionDefinition, *def2.ActionDefinition)
				}
			}
		}
	}

}
