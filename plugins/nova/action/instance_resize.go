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

type InstanceResizeInput struct {
	WorkflowId   string                 `json:"workflow_id"`
	UserAuthorty map[string]interface{} `json:"auth" validate:"requiredInMap: tenantid, token,"`
	InstanceUUID string                 `json:"uuid" validate:"required"`
	Body         string                 `json:"body"`
	Sleep        int                    `json:"sleep"`
}

type InstanceResize struct {
	attributes map[string]interface{}
	ctx        *contextx.Context
}

func (s *InstanceResize) Initialize(attrs map[string]interface{}) pluginx.EndpointCallable {
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
	return &InstanceResize{attributes: attrs, ctx: context}
}

func (s *InstanceResize) Run(input InstanceResizeInput) (interface{}, error) {
	log.Debugf(s.ctx, "Instance Resize got input [%v]. The correlation information [%v] ", input, s.attributes)
	var obj_result objects.ActionResult

	if err := common.ValidateStruct(input); len(err) > 0 {
		var err_str string
		for _, value := range err {
			err_str += value.Error() + ";"
		}
		obj_result.Err = "Validate Error"
		obj_result.Data = err_str
		return &obj_result, nil
	}

	time.Sleep(time.Duration(input.Sleep) * time.Second)
	bodyMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(input.Body), &bodyMap)
	if err != nil {
		obj_result.Err = "Body to Map Error"
		obj_result.Data = err.Error()
		return &obj_result, nil
	}

	url := fmt.Sprintf("%v/%v/servers/%v/action", URL, input.UserAuthorty["tenantid"], input.InstanceUUID)
	body_json, _ := json.Marshal(map[string]interface{}{"migrate": bodyMap})
	body := string(body_json)
	token := input.UserAuthorty["token"]

	resp, code := NovaRunAPI(s.ctx, "POST", url, token.(string), body, input.WorkflowId)
	if code >= 400 {
		obj_result.Err = "Request NOVA API Error"
		obj_result.Data = fmt.Sprintf("StatusCode:%v, Response:%v", code, resp)
	} else {
		obj_result.Data = fmt.Sprintf("StatusCode:%v, Response:%v", code, resp)
	}
	return &obj_result, nil
}
