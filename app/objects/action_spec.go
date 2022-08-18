package objects

import (
	"encoding/json"
	"reflect"
	"seraph/pkg/log"
)

type ActionSpec struct {
	Name      string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Base      string                 `yaml:"base,omitempty" json:"base,omitempty"`
	BaseInput map[string]interface{} `yaml:"base-input,omitempty" json:"base_input,omitempty"`
	Input     []interface{}          `yaml:"input,omitempty" json:"input,omitempty"`
	Output    map[string]interface{} `yaml:"output,omitempty" json:"output,omitempty"`

	base       string
	baseInput  map[string]interface{}
	inputNames []string
}

func (s *ActionSpec) GetMergedInput(input map[string]interface{}) map[string]interface{} {
	mergedInput := map[string]interface{}{}
	for k, v := range input {
		mergedInput[k] = v
	}

	if s.Input != nil {
		for _, v := range s.Input {
			switch reflect.ValueOf(v).Kind() {
			case reflect.Map:
				for k, vv := range v.(map[string]interface{}) {
					if _, ok := mergedInput[k]; !ok {
						mergedInput[k] = vv
					}
				}
			}
		}
	}
	return mergedInput
}

func (s *ActionSpec) initializeInput() error {
	s.inputNames = []string{}

	if s.Input != nil {
		for _, v := range s.Input {
			switch reflect.ValueOf(v).Kind() {
			case reflect.String:
				s.inputNames = append(s.inputNames, v.(string))
			case reflect.Map:
				for k := range v.(map[string]interface{}) {
					s.inputNames = append(s.inputNames, k)
				}
			}
		}
	}
	return nil
}

func (s *ActionSpec) initializeCommand() error {
	var err error
	var params = map[string]interface{}{}

	s.base, params, err = ParseCMDAndInputs(s.Base)
	if err != nil {
		return err
	}

	if (s.BaseInput != nil && len(s.BaseInput) > 0) || (params != nil && len(params) > 0) {
		s.baseInput = map[string]interface{}{}
	}

	if s.BaseInput != nil {
		for k, v := range s.BaseInput {
			s.baseInput[k] = v
		}
	}

	for k, v := range params {
		s.baseInput[k] = v
	}
	return nil
}

func (s *ActionSpec) Initialize() error {
	if err := s.initializeCommand(); err != nil {
		return err
	}

	if err := s.initializeInput(); err != nil {
		return err
	}

	return nil
}

func (s *ActionSpec) GetBase() string {
	return s.base
}

func (s *ActionSpec) GetBaseInput() map[string]interface{} {
	return s.baseInput
}

func (s *ActionSpec) GetInputNames() []string {
	return s.inputNames
}

func (s *ActionSpec) ToString() string {
	result, err := json.Marshal(s)
	if err != nil {
		log.Debugf(nil, "to string failed, error: %s", err.Error())
	}
	return string(result)
}
