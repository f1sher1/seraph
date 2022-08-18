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

var (
	KDFS = "KDFS" // 网盘统称
	SSD  = "SSD"  //本地数据盘
)

type CheckInstanceStorageTypeInput struct {
	WorkflowId   string                 `json:"workflow_id"`
	UserAuthorty map[string]interface{} `json:"auth" validate:"requiredInMap: tenantid, token,"`
	InstanceUUID string                 `json:"uuid" validate:"required"`
	Body         map[string]interface{} `json:"body"`
	Sleep        int                    `json:"sleep"`
	Retry        int                    `json:"retry"`
	Target       string                 `json:"target"`
}

type CheckInstanceStorageType struct {
	attributes map[string]interface{}
	ctx        *contextx.Context
}

func (s *CheckInstanceStorageType) Initialize(attrs map[string]interface{}) pluginx.EndpointCallable {
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
	return &CheckInstanceStorageType{attributes: attrs, ctx: context}
}

func (s *CheckInstanceStorageType) Run(input CheckInstanceStorageTypeInput) (interface{}, error) {
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
		var storageType string
		data, code = NovaRunAPI(s.ctx, "GET", url, token.(string), "", input.WorkflowId)
		if code >= 400 {
			obj_result.Data = fmt.Sprintf("StatusCode: %v, Response: %v", code, data)
			obj_result.Err = "Request NOVA API Error"
			break
		}
		if input.Target != "" {
			res := make(map[string]map[string]interface{})
			if err := json.Unmarshal([]byte(data), &res); err != nil {
				obj_result.Data = fmt.Sprintf("[%v], Origain Data: %v", err.Error(), data)
				obj_result.Err = "To Format Map Error"
				break
			}
			if flavor, ok := res["server"]["flavor"]; !ok {
				obj_result.Data = fmt.Sprintf("No find the instance flavor, the instance is %v", input.InstanceUUID)
				obj_result.Err = "Get Instance Detail Error"
				break
			} else {
				switch t := flavor.(type) {
				case map[string]interface{}:
					if ephemeralGb, ok := t["ephemeral_gb"]; !ok {
						storageType = KDFS
					} else {
						switch m := ephemeralGb.(type) {
						case int:
							if m > 0 {
								storageType = KDFS
							} else {
								storageType = KDFS
							}
						}
					}
				}
			}
			log.Debugf(s.ctx, "The instance storage type is [%v], want type [%v]", storageType, input.Target)
			if storageType == input.Target {
				obj_result.Data = fmt.Sprintf("StatusCode: %v, Response: %v", code, data)
				obj_result.Err = ""
				break
			} else {
				obj_result.Err = fmt.Sprintf("The loop ends, the instance storage type is %v, not %v", storageType, input.Target)
			}
		} else {
			obj_result.Data = fmt.Sprintf("StatusCode: %v, Response: %v", code, data)
		}
	}
	return &obj_result, nil
}
