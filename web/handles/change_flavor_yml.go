package handles

import (
	"encoding/json"
	"seraph/app/objects"
	"seraph/pkg/contextx"

	"github.com/google/uuid"
)

func ChangeFlavorYmlHandle(ctx *contextx.Context, token, instance_uuid, tenantid string, map_body map[string]interface{}) (int, string) {
	var autoRestart bool
	body, ok := map_body["ColdMigrateBody"]
	if !ok {
		return 404, "Parameter ColdMigrateBody is missing"
	} else {
		switch t := body.(type) {
		case map[string]interface{}:
			if len(t) == 0 {
				return 404, "Parameter ColdMigrateBody cannot use {}"
			} else {
				b, _ := json.Marshal(t)
				body = string(b)
			}
		default:
			return 404, "Parameter ColdMigrateBody cannot use null"
		}
	}
	auto, ok := map_body["AutoRestart"]
	if !ok {
		return 404, "Parameter AutoRestart is missing"
	} else {
		switch t := auto.(type) {
		case bool:
			autoRestart = t
		default:
			return 404, "Parameter AutoRestart type must be bool"
		}
	}

	// 初始化wf execution Input
	exID := uuid.NewString()
	ctx.Set("workflow", exID)
	input := objects.Table{
		"instance_uuid": instance_uuid,
		"tenantid":      tenantid,
		"token":         token,
		"workflow_id":   exID,
		"body":          body,
		"force_restart": true,
	}
	var ymlPath string
	if autoRestart {
		ymlPath = "./web/yaml/change_instance_flavor.yml"
	} else {
		ymlPath = "./web/yaml/change_instance_flavor_nostart.yml"
	}
	wfDef, err := parseYamlFileToWorkflowDefinition(ctx, ymlPath)
	if err != nil {
		return 500, err.Error()
	}

	Runner(ctx, wfDef, exID, input)

	return 200, exID
}
