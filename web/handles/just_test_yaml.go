package handles

import (
	"github.com/google/uuid"
	"seraph/app/objects"
	"seraph/pkg/contextx"
)

func JustTest(ctx *contextx.Context, token, instance_uuid, tenantid string, map_body map[string]interface{}) (int, string) {
	// todo add something to check the request
	exID := uuid.NewString()
	ctx.Set("workflow", exID)
	input := objects.Table{
		"instance_uuid": instance_uuid,
		"tenantid":      tenantid,
		"token":         token,
		"workflow_id":   exID,
	}
	ymlPath := "./web/yaml/test.yml"

	wfDef, err := parseYamlFileToWorkflowDefinition(ctx, ymlPath)
	if err != nil {
		return 500, err.Error()
	}

	Runner(ctx, wfDef, exID, input)

	return 200, exID
}
