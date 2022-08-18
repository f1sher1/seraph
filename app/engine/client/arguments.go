package client

import (
	"encoding/json"
	"reflect"
	"seraph/app/objects"
)

type StartWorkflowArg struct {
	WorkflowID          string                 `json:"wf_id"`
	WorkflowNamespace   string                 `json:"wf_namespace"`
	WorkflowExecutionID string                 `json:"wf_ex_id"`
	WorkflowInput       map[string]interface{} `json:"wf_input"`
	Params              map[string]interface{} `json:"params"`
	Description         string                 `json:"description"`
}

type OnActionCompleteArg struct {
	ActionExecutionID string      `json:"action_ex_id"`
	Result            interface{} `json:"result"`
	IsWorkflowAction  bool        `json:"wf_action"`
}

func (arg OnActionCompleteArg) GetResult() *objects.ActionResult {
	switch reflect.ValueOf(arg.Result).Kind() {
	case reflect.Map:
		res := &objects.ActionResult{}
		jsonStr, err := json.Marshal(arg.Result)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(jsonStr, res)
		if err != nil {
			panic(err)
		}
		return res
	default:
		return arg.Result.(*objects.ActionResult)
	}
}
