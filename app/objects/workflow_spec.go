package objects

import "reflect"

type WorkflowSpec struct {
	Name         string              `yaml:"name,omitempty" json:"name,omitempty"`
	Description  string              `yaml:"description,omitempty" json:"description,omitempty"`
	Tags         []string            `yaml:"tags" json:"tags,omitempty"`
	Type         string              `yaml:"type,omitempty" json:"type"`
	Inputs       []interface{}       `yaml:"input,omitempty" json:"inputs"`
	Tasks        map[string]TaskSpec `yaml:"tasks" json:"tasks"`
	TaskDefaults *TaskSpec           `yaml:"task-defaults,omitempty" json:"task_defaults"`
	Target       string              `yaml:"target" json:"target,omitempty"`

	inputName []string
}

func (w *WorkflowSpec) Initialize() error {
	if w.TaskDefaults != nil {
		if err := w.TaskDefaults.Initialize(); err != nil {
			return err
		}
	}

	for taskName, taskSpec := range w.Tasks {
		if err := taskSpec.Initialize(); err != nil {
			return err
		}
		if taskSpec.Name == "" {
			taskSpec.Name = taskName
		}
		taskSpec.SetDefaults(w.TaskDefaults)
		w.Tasks[taskName] = taskSpec
	}

	if w.Inputs != nil {
		for _, input := range w.Inputs {
			switch reflect.ValueOf(input).Kind() {
			case reflect.String:
				w.inputName = append(w.inputName, input.(string))
			case reflect.Map:
				for k := range input.(map[string]interface{}) {
					w.inputName = append(w.inputName, k)
				}
			}
		}
	}

	return nil
}

func (w WorkflowSpec) GetInputNames() []string {
	return w.inputName
}

func (w *WorkflowSpec) HasOnErrorClause(name string) bool {
	if len(w.Tasks[name].GetOnErrorClause()) > 0 {
		return true
	} else {
		if w.TaskDefaults != nil && len(w.TaskDefaults.GetOnErrorClause()) > 0 {
			return true
		}
		return false
	}
}
