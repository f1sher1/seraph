package objects

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"seraph/app/workflow/states"
	"seraph/pkg/log"
)

type TaskRetrySpec struct {
	Count   int    `yaml:"count,omitempty" json:"count,omitempty"`
	Delay   int    `yaml:"delay,omitempty" json:"delay,omitempty"`
	BreakOn string `yaml:"break-on,omitempty" json:"break_on,omitempty"`
}

type ParentResult struct {
	IsGetParentResult bool   `yaml:"is-getparentresult,omitempty" json:"is_getparentresult,omitempty"`
	Result            string `yaml:"result,omitempty" json:"result,omitempty"`
}

type TaskSpec struct {
	Name            string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Description     string                 `yaml:"description,omitempty" json:"description"`
	Workflow        string                 `yaml:"workflow,omitempty" json:"workflow,omitempty"`
	Action          string                 `yaml:"action,omitempty" json:"action,omitempty"`
	Parameters      []interface{}          `yaml:"params,omitempty" json:"parameters"`
	Retry           TaskRetrySpec          `yaml:"retry,omitempty" json:"retry,omitempty"`
	WithItems       interface{}            `yaml:"with-items,omitempty" json:"with_items,omitempty"`
	WaitBefore      int                    `yaml:"wait-before,omitempty" json:"wait_before"`
	WaitAfter       int                    `yaml:"wait-after,omitempty" json:"wait_after"`
	Concurrency     int                    `yaml:"concurrency,omitempty" json:"concurrency"`
	OnComplete      []interface{}          `yaml:"on-complete,omitempty" json:"on_complete,omitempty"`
	OnSuccess       []interface{}          `yaml:"on-success,omitempty" json:"on_success,omitempty"`
	OnError         []interface{}          `yaml:"on-error,omitempty" json:"on_error,omitempty"`
	Requires        []string               `yaml:"requires,omitempty" json:"requires,omitempty"`
	Publish         map[string]interface{} `yaml:"publish,omitempty" json:"publish,omitempty"`
	Target          string                 `yaml:"target,omitempty" json:"target,omitempty"`
	Join            interface{}            `yaml:"join,omitempty" json:"join,omitempty"`
	Timeout         int                    `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	GetParentResult ParentResult           `yaml:"get-parent,omitempty" json:"get_parent,omitempty"`

	workflowName string
	actionName   string
	params       []interface{}
	withItems    map[string]interface{}

	onComplete map[string]string
	onSuccess  map[string]string
	onError    map[string]string

	defaults *TaskSpec
}

func (t *TaskSpec) GetType() string {
	if t.workflowName != "" {
		return "WORKFLOW"
	}
	return "ACTION"
}

func (t *TaskSpec) initializeCommand() error {
	var err error
	var params = map[string]interface{}{}

	if t.Workflow != "" {
		t.workflowName, params, err = ParseCMDAndInputs(t.Workflow)
	} else if t.Action != "" {
		t.actionName, params, err = ParseCMDAndInputs(t.Action)
	}

	if err != nil {
		return err
	}

	if (t.Parameters != nil && len(t.Parameters) > 0) || (params != nil && len(params) > 0) {
		t.params = []interface{}{}
	}

	if t.Parameters != nil {
		for _, parameter := range t.Parameters {
			t.params = append(t.params, parameter)
		}
	}

	for name, value := range params {
		p := map[string]interface{}{}
		p[name] = value
		t.params = append(t.params, p)
	}

	return nil
}

func (t *TaskSpec) initializeWithItems() error {
	var err error
	if t.WithItems != nil {
		t.withItems = map[string]interface{}{}

		var withItemSlice []string
		tTypeKind := reflect.ValueOf(t.WithItems).Type().Kind()

		switch tTypeKind {
		case reflect.String:
			withItemSlice = append(withItemSlice, t.WithItems.(string))
		case reflect.Slice:
			withItemSliceInterface, ok := t.WithItems.([]interface{})
			if !ok {
				return errors.New("invalid list with-items")
			}
			for _, i := range withItemSliceInterface {
				withItemSlice = append(withItemSlice, i.(string))
			}
		default:
			return fmt.Errorf("invalid with-items type '%s'", tTypeKind)
		}

		t.withItems, err = MatchWithItems(withItemSlice)
		if err != nil {
			return fmt.Errorf("invalid with-items, error: %s", err.Error())
		}
	}
	return nil
}

func (t *TaskSpec) initializeTransition() error {
	t.onComplete = t.parseTransition(t.OnComplete)
	t.onSuccess = t.parseTransition(t.OnSuccess)
	t.onError = t.parseTransition(t.OnError)
	return nil
}

func (t *TaskSpec) parseTransition(transition []interface{}) map[string]string {
	parsedTrans := map[string]string{}
	for _, trans := range transition {
		v := reflect.ValueOf(trans)
		switch v.Kind() {
		case reflect.String:
			parsedTrans[trans.(string)] = ""
		case reflect.Map:
			for name, value := range trans.(map[string]interface{}) {
				parsedTrans[name] = value.(string)
			}
		}
	}
	return parsedTrans
}

func (t *TaskSpec) Initialize() error {
	var err error
	err = t.initializeCommand()
	if err != nil {
		return err
	}

	err = t.initializeWithItems()
	if err != nil {
		return err
	}

	err = t.initializeTransition()
	if err != nil {
		return err
	}

	return nil
}

func (t *TaskSpec) SetDefaults(defaults *TaskSpec) {
	t.defaults = defaults
}

func (t TaskSpec) GetWorkflowName() string {
	return t.workflowName
}

func (t TaskSpec) GetActionName() string {
	return t.actionName
}

func (t TaskSpec) GetOnSuccessClause() map[string]string {
	if len(t.onSuccess) > 0 {
		return t.onSuccess
	} else {
		if t.defaults != nil && t.onSuccess != nil {
			return t.defaults.onSuccess
		}
	}
	return map[string]string{}
}

func (t TaskSpec) GetOnErrorClause() map[string]string {
	if len(t.onError) > 0 {
		return t.onError
	} else {
		if t.defaults != nil && t.onError != nil {
			return t.defaults.onError
		}
	}
	return map[string]string{}
}

func (t TaskSpec) GetOnCompleteClause() map[string]string {
	if len(t.onComplete) > 0 {
		return t.onComplete
	} else {
		if t.defaults != nil && t.onComplete != nil {
			return t.defaults.onComplete
		}
	}
	return map[string]string{}
}

func (t *TaskSpec) ToString() string {
	content, err := json.Marshal(t)
	if err != nil {
		log.Errorf(nil, "spec '%v' marshal failed, error: %s", t, err.Error())
		return ""
	}

	return string(content)
}

func (t TaskSpec) GetParameters() map[string]interface{} {
	params := map[string]interface{}{}
	for _, p := range t.params {
		switch reflect.ValueOf(p).Kind() {
		case reflect.Map:
			for k, v := range p.(map[string]interface{}) {
				params[k] = v
			}
		case reflect.String:
			params[p.(string)] = nil
		}
	}
	return params
}

func (t TaskSpec) GetWithItems() map[string]interface{} {
	return t.withItems
}

func (t *TaskSpec) GetPublish(state string) *PublishSpec {
	if states.IsSuccess(state) && len(t.Publish) > 0 {
		return NewPublishSpec(t.Publish, nil)
	}
	// todo; add publish on error
	return nil
}
