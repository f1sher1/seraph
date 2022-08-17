package client

import (
	"encoding/json"
	"log"
	"seraph/pkg/contextx"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_StartWorkflow(t *testing.T) {
	asserter := assert.New(t)

	ctx := contextx.NewContext()
	ctx.Set("project_id", "test-project")

	clt, err := NewClient()
	if asserter.NoError(err) {
		err = clt.Initialize()
		if asserter.NoError(err) {
			ex, err := clt.StartWorkflow(ctx, "workflow_id_identifier", "namespace", "ex_id", map[string]interface{}{"a": "1"}, "description content", nil)
			if asserter.NoError(err) {
				if asserter.Equal("workflow_id_identifier", ex.WorkflowID) {
					if asserter.Equal("namespace", ex.WorkflowNamespace) {
						if asserter.Equal("ex_id", ex.ID) {
							if asserter.Equal(map[string]interface{}{"a": "1"}, map[string]interface{}(ex.Input)) {
								if asserter.Equal("description content", ex.Description) {
									log.Printf("%v", ex)
									jsonBytes, _ := json.Marshal(ex)
									log.Printf("%s", jsonBytes)
								}
							}
						}
					}
				}
			}
		}
	}
}
