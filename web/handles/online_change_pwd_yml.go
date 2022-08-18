package handles

import (
	"seraph/app/objects"
	"seraph/pkg/contextx"

	"github.com/google/uuid"
)

func OnlineChangePwdYmlHandle(ctx *contextx.Context, token, instance_uuid, tenantid string, map_body map[string]interface{}) (int, string) {
	var newPwd string
	var forceRestart bool
	newpwd, ok := map_body["adminPass"]
	if !ok {
		return 404, "Parameter adminPass is missing"
	} else {
		switch t := newpwd.(type) {
		case string:
			if t == "" {
				return 404, "Parameter adminPass cannot use \"\""
			}
			newPwd = t
		default:
			return 404, "Parameter adminPass type must be string"
		}
	}
	force, ok := map_body["ForceRestart"]
	if !ok {
		return 404, "Parameter ForceRestart is missing"
	} else {
		switch t := force.(type) {
		case bool:
			forceRestart = t
		default:
			return 404, "Parameter ForceRestart type must be bool"
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
		"pwd":           newPwd,
		"force_restart": forceRestart,
	}
	ymlPath := "./web/yaml/online_change_pwd.yml"

	wfDef, err := parseYamlFileToWorkflowDefinition(ctx, ymlPath)
	if err != nil {
		return 500, err.Error()
	}

	Runner(ctx, wfDef, exID, input)

	return 200, exID
}
