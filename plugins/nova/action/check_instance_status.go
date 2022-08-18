package action

import (
	"encoding/json"
	"fmt"
	"seraph/app/objects"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"seraph/pkg/pluginx"
	"seraph/plugins/nova/common"
	"time"
)

type CheckInstanceStatusInput struct {
	WorkflowId   string                 `json:"workflow_id"`
	UserAuthorty map[string]interface{} `json:"auth" validate:"requiredInMap: tenantid, token,"`
	InstanceUUID string                 `json:"uuid" validate:"required"`
	Body         map[string]interface{} `json:"body"`
	Sleep        int                    `json:"sleep"`
	Retry        int                    `json:"retry"`
	LastStatus   string                 `json:"last_status"`
}

type CheckInstanceStatus struct {
	attributes map[string]interface{}
	ctx        *contextx.Context
}

func (s *CheckInstanceStatus) Initialize(attrs map[string]interface{}) pluginx.EndpointCallable {
	//  初始化context
	context := contextx.NewContext()
	if value, ok := attrs["context"]; !ok {
		context.Set("requestId", "*")
		context.Set("workflow", "*")
	} else {
		switch t := value.(type) {
		case map[string]interface{}:
			for k, v := range t {
				context.Set(k, v)
			}
		}
	}
	return &CheckInstanceStatus{attributes: attrs, ctx: context}
}

func (s *CheckInstanceStatus) Run(input CheckInstanceStatusInput) (interface{}, error) {
	log.Debugf(s.ctx, "Instance Detail got input [%v]. The correlation information [%v] ", input, s.attributes)
	var obj_result objects.ActionResult
	// 初始化入参
	if input.Retry < 1 {
		input.Retry = 1
	}

	if err := common.ValidateStruct(input); len(err) > 0 {
		var err_str string
		for _, value := range err {
			err_str += value.Error() + ";"
		}
		obj_result.Err = "Validate Error"
		obj_result.Data = err_str
		return &obj_result, nil
	}
	for i := 0; i < input.Retry; i++ {
		time.Sleep(time.Duration(input.Sleep) * time.Second)

		url := fmt.Sprintf("%v/%v/servers/%v", URL, input.UserAuthorty["tenantid"], input.InstanceUUID)
		token := input.UserAuthorty["token"]

		var code int
		var data string
		data, code = NovaRunAPI(s.ctx, "GET", url, token.(string), "", input.WorkflowId)
		if code >= 400 {
			obj_result.Data = fmt.Sprintf("StatusCode: %v, Response: %v", code, data)
			obj_result.Err = "Request NOVA API Error"
			break
		}
		if input.LastStatus != "" {
			res := make(map[string]map[string]interface{})
			if err := json.Unmarshal([]byte(data), &res); err != nil {
				obj_result.Data = fmt.Sprintf("[%v], Origain Data: %v", err.Error(), data)
				obj_result.Err = "To Format Map Error"
				break
			}
			log.Debugf(s.ctx, "$$$ %#v, %#v", res["server"]["status"], res["server"]["OS-EXT-STS:task_state"])
			if res["server"]["status"] == input.LastStatus && res["server"]["OS-EXT-STS:task_state"] == nil {
				obj_result.Data = fmt.Sprintf("StatusCode: %v, Response: %v", code, data)
				obj_result.Err = ""
				break
			} else {
				obj_result.Err = fmt.Sprintf("The loop ends, the final status is %v task_status is %v, not %v, please operate manually.", res["server"]["status"], res["server"]["OS-EXT-STS:task_state"], input.LastStatus)
			}
		} else {
			obj_result.Data = fmt.Sprintf("StatusCode: %v, Response: %v", code, data)
		}
	}
	return &obj_result, nil
}
