package action

import (
	"fmt"
	"seraph/app/objects"
	"seraph/app/unimq"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"seraph/pkg/pluginx"
	"seraph/plugins/nova/common"
	"time"

	"github.com/google/uuid"
)

// Body:{workflow_id:, name:, status:, instance_id:, }
type SendUniMQInput struct {
	UserAuthorty map[string]interface{} `json:"auth" validate:"requiredInMap: tenantid, token,"`
	Body         map[string]interface{} `json:"body"`
	Sleep        int                    `json:"sleep"`
	Retry        int                    `json:"retry"`
}

type SendUniMQ struct {
	attributes map[string]interface{}
	ctx        *contextx.Context
}

func (s *SendUniMQ) Initialize(attrs map[string]interface{}) pluginx.EndpointCallable {
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
	return &SendUniMQ{attributes: attrs, ctx: context}
}

func (s *SendUniMQ) Run(input SendUniMQInput) (interface{}, error) {
	log.Debugf(s.ctx, "Send UniMQ got input [%v]. The correlation information [%v] ", input, s.attributes)
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
	var requestId string
	if rId, ok := s.ctx.GetMap()["requestId"]; !ok {
		requestId = "****"
	} else {
		requestId = rId.(string)
	}
	for i := 0; i < input.Retry; i++ {
		now := time.Now()
		msg := unimq.Msg{
			ContextRequestId:     requestId,
			ContextRequestAction: input.Body["name"].(string),
			ContextRoles:         []string{"admin"},
			ContextIsAdmin:       true,
			ContextReadDeleted:   "no",
			ContextProjectId:     input.UserAuthorty["tenantid"].(string),
			EventType:            fmt.Sprintf("seraph.%v.%v", input.Body["name"], input.Body["status"]),
			Priority:             "INFO",
			Payload:              input.Body,
			Timestamp:            now.Format("2006-01-01T15:04:05.000"),
			UniqueId:             fmt.Sprintf("wf-%v", uuid.NewString()),
			WorkflowId:           input.Body["workflow_id"].(string),
		}
		log.Debugf(s.ctx, "notify: %#v", msg)
		resp, code := unimq.DeliverMessage(s.ctx, msg)
		if code >= 400 {
			obj_result.Err = "Request UniMQ API Error"
			obj_result.Data = fmt.Sprintf("StatusCode: %v, Request: %v", code, resp)
		} else {
			obj_result.Err = ""
			obj_result.Data = fmt.Sprintf("StatusCode: %v, Request: %v", code, resp)
			break
		}
	}
	return &obj_result, nil
}
