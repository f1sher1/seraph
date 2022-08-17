package handles

import (
	"encoding/json"
	"seraph/app/objects"
	"seraph/app/workflow"
	"seraph/pkg/contextx"
	"seraph/pkg/log"

	"github.com/google/uuid"
)

func ChangeFlavorHandle(ctx *contextx.Context, token, instance_uuid, tenantid string, map_body map[string]interface{}) (int, string) {
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
				body = t
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
	input := objects.Table{
		"instance_uuid": instance_uuid,
		"tenantid":      tenantid,
		"token":         token,
	}
	exID := uuid.NewString()
	ctx.Set("workflow", exID)
	wfSpec := objects.WorkflowSpec{}
	if autoRestart {
		wfSpec = objects.WorkflowSpec{
			Name:        "{{ .instance_uuid }}",
			Description: "desc",
			Tags:        nil,
			Type:        "direct",
			Tasks: map[string]objects.TaskSpec{
				"check_instance_is_on": {
					Name:   "check_instance_is_on",
					Action: "nova.check_instance_status",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"retry":       1,
							"sleep":       0,
							"last_status": "ACTIVE",
						},
					},
					OnSuccess: []interface{}{
						"instance_stop",
					},
					OnError: []interface{}{
						"check_instance_is_off",
					},
				},
				"check_instance_is_off": {
					Name:   "check_instance_is_off",
					Action: "nova.check_instance_status",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"retry":       1,
							"sleep":       0,
							"last_status": "SHUTOFF",
						},
					},
					OnSuccess: []interface{}{
						"instance_resize",
					},
					OnError: []interface{}{
						"send_unimq_error",
					},
				},
				"instance_stop": {
					Name:   "instance_stop",
					Action: "nova.instance_stop",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"body": map[string]interface{}{
								"force_stop": true,
							},
						},
					},
					OnSuccess: []interface{}{
						"get_instance_status_stop",
					},
					OnError: []interface{}{
						"send_unimq_error",
					},
				},
				"get_instance_status_stop": {
					Name:   "get_instance_status_stop",
					Action: "nova.check_instance_status",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"retry":       300,
							"sleep":       2,
							"last_status": "SHUTOFF",
						},
					},
					OnSuccess: []interface{}{
						"instance_resize",
					},
					OnError: []interface{}{
						"send_unimq_error",
					},
				},
				"instance_resize": {
					Name:   "instance_resize",
					Action: "nova.instance_resize",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"body": body,
						},
					},
					OnSuccess: []interface{}{
						"get_instance_status_resize",
					},
					OnError: []interface{}{
						"send_unimq_error",
					},
				},
				"get_instance_status_resize": {
					Name:   "get_instance_status_resize",
					Action: "nova.check_instance_status",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"retry":       1800,
							"sleep":       5,
							"last_status": "SHUTOFF",
						},
					},
					OnSuccess: []interface{}{
						"instance_start",
					},
					OnError: []interface{}{
						"send_unimq_error",
					},
				},
				"instance_start": {
					Name:   "instance_start",
					Action: "nova.instance_start",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
						},
					},
					OnSuccess: []interface{}{
						"send_unimq_success",
					},
					OnError: []interface{}{
						"send_unimq_error",
					},
				},
				"send_unimq_success": {
					Name:   "send_unimq_success",
					Action: "nova.send_unimq",
					Parameters: []interface{}{
						map[string]interface{}{
							"uuid": "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"body": map[string]interface{}{
								"workflow_id":   exID,
								"name":          "wf-change_flavor",
								"status":        "end",
								"instance_uuid": "{{ .instance_uuid }}",
							},
						},
					},
				},
				"send_unimq_error": {
					Name:            "send_unimq_error",
					Action:          "nova.send_unimq",
					GetParentResult: objects.ParentResult{IsGetParentResult: true},
					Parameters: []interface{}{
						map[string]interface{}{
							"uuid": "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"body": map[string]interface{}{
								"workflow_id":   exID,
								"name":          "wf-change_flavor",
								"status":        "failed",
								"instance_uuid": "{{ .instance_uuid }}",
								"msg":           "{{ .parent_result }}",
							},
						},
					},
				},
			},
			TaskDefaults: nil,
		}
	} else {
		wfSpec = objects.WorkflowSpec{
			Name:        "{{ .instance_uuid }}",
			Description: "desc",
			Tags:        nil,
			Type:        "direct",
			Tasks: map[string]objects.TaskSpec{
				"check_instance_is_on": {
					Name:   "check_instance_is_on",
					Action: "nova.check_instance_status",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"retry":       1,
							"sleep":       0,
							"last_status": "ACTIVE",
						},
					},
					OnSuccess: []interface{}{
						"instance_stop",
					},
					OnError: []interface{}{
						"check_instance_is_off",
					},
				},
				"check_instance_is_off": {
					Name:   "check_instance_is_off",
					Action: "nova.check_instance_status",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"retry":       1,
							"sleep":       0,
							"last_status": "SHUTOFF",
						},
					},
					OnSuccess: []interface{}{
						"instance_resize",
					},
					OnError: []interface{}{
						"send_unimq_error",
					},
				},
				"instance_stop": {
					Name:   "instance_stop",
					Action: "nova.instance_stop",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"body": map[string]interface{}{
								"force_stop": true,
							},
						},
					},
					OnSuccess: []interface{}{
						"get_instance_status_stop",
					},
					OnError: []interface{}{
						"send_unimq_error",
					},
				},
				"get_instance_status_stop": {
					Name:   "get_instance_status_stop",
					Action: "nova.check_instance_status",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"retry":       300,
							"sleep":       2,
							"last_status": "SHUTOFF",
						},
					},
					OnSuccess: []interface{}{
						"instance_resize",
					},
					OnError: []interface{}{
						"send_unimq_error",
					},
				},
				"instance_resize": {
					Name:   "instance_resize",
					Action: "nova.instance_resize",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"body": body,
						},
					},
					OnSuccess: []interface{}{
						"get_instance_status_resize",
					},
					OnError: []interface{}{
						"send_unimq_error",
					},
				},
				"get_instance_status_resize": {
					Name:   "get_instance_status_resize",
					Action: "nova.check_instance_status",
					Parameters: []interface{}{
						map[string]interface{}{
							"workflow_id": exID,
							"uuid":        "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"retry":       1800,
							"sleep":       5,
							"last_status": "SHUTOFF",
						},
					},
					OnSuccess: []interface{}{
						"send_unimq_success",
					},
					OnError: []interface{}{
						"send_unimq_error",
					},
				},
				"send_unimq_success": {
					Name:   "send_unimq_success",
					Action: "nova.send_unimq",
					Parameters: []interface{}{
						map[string]interface{}{
							"uuid": "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"body": map[string]interface{}{
								"workflow_id":   exID,
								"name":          "wf-change_flavor",
								"status":        "end",
								"instance_uuid": "{{ .instance_uuid }}",
							},
						},
					},
				},
				"send_unimq_error": {
					Name:            "send_unimq_error",
					Action:          "nova.send_unimq",
					GetParentResult: objects.ParentResult{IsGetParentResult: true},
					Parameters: []interface{}{
						map[string]interface{}{
							"uuid": "{{ .instance_uuid }}",
							"auth": map[string]string{
								"tenantid": "{{ .tenantid }}",
								"token":    "{{ .token }}",
							},
							"body": map[string]interface{}{
								"workflow_id":   exID,
								"name":          "wf-change_flavor",
								"status":        "failed",
								"instance_uuid": "{{ .instance_uuid }}",
								"msg":           "{{ .parent_result }}",
							},
						},
					},
				},
			},
			TaskDefaults: nil,
		}
	}

	wfDef := objects.NewWorkflowDefinition()
	specBytes, err := json.Marshal(wfSpec)
	if err != nil {
		log.Errorf(exID, "WorkflowSpec struct change JSON error [%v]", err.Error())
		return 500, err.Error()
	}
	wfDef.Spec = string(specBytes)

	s, _ := wfDef.GetSpec()

	wf := workflow.Workflow{
		Spec: s,
		Ctx:  ctx,
	}
	wf.Initialize()

	wf.Start(exID, "", "", nil, input, nil)
	return 200, exID
}
