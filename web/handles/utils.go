package handles

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"seraph/app/objects"
	"seraph/app/workflow"
	"seraph/pkg/contextx"
	"seraph/pkg/log"

	"gopkg.in/yaml.v2"
)

func changeMap(m map[interface{}]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range m {
		switch t := k.(type) {
		case interface{}:
			switch _t := v.(type) {
			case map[interface{}]interface{}:
				res[fmt.Sprint(t)] = changeMap(_t)
			default:
				res[fmt.Sprint(t)] = _t
			}
		default:
			switch _t := v.(type) {
			case map[interface{}]interface{}:
				res[fmt.Sprint(t)] = changeMap(_t)
			default:
				res[fmt.Sprint(t)] = _t
			}
		}
	}
	return res
}

func parseYamlFileToWorkflowDefinition(ctx *contextx.Context, ymlPath string) (*objects.WorkflowDefinition, error) {
	wfSpec := objects.WorkflowSpec{}
	wfDef := objects.NewWorkflowDefinition()
	fd, err := ioutil.ReadFile(ymlPath)
	if err != nil {
		log.Errorf(ctx, "Read YAML file error [%v]", err.Error())
		return wfDef, err
	}
	err = yaml.Unmarshal(fd, &wfSpec)
	if err != nil {
		log.Errorf(ctx, "YAML file change WorkflowSpec struct error [%v]", err.Error())
		return wfDef, err
	}
	// 转换map[insterface{}]interface{}
	for _, taskspec := range wfSpec.Tasks {
		for i, value := range taskspec.Parameters {
			switch val := value.(type) {
			case map[interface{}]interface{}:
				newValue := changeMap(val)
				taskspec.Parameters[i] = newValue
			}
		}
	}

	specBytes, err := json.Marshal(wfSpec)
	if err != nil {
		log.Errorf(ctx, "WorkflowSpec struct change JSON error [%v]", err.Error())
		return wfDef, err
	}
	wfDef.Spec = string(specBytes)
	return wfDef, nil
}

func Runner(ctx *contextx.Context, wfDef *objects.WorkflowDefinition, exID string, input objects.Table) {
	s, _ := wfDef.GetSpec()
	wf := workflow.Workflow{
		Spec: s,
		Ctx:  ctx,
	}
	wf.Initialize()

	wf.Start(exID, "", "", nil, input, nil)
}
