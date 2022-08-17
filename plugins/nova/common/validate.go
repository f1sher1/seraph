package common

import (
	"fmt"
	"reflect"
	"strings"
)

const tagName = "validate"

type Validator interface {
	Validate(interface{}) (bool, error)
}

type DefaultValidator struct{}

func (v DefaultValidator) Validate(val interface{}) (bool, error) {
	return true, nil
}

type ErrorValidator struct{ Message string }

func (v ErrorValidator) Validate(val interface{}) (bool, error) {
	return false, fmt.Errorf(v.Message)
}

type RequiredValidator struct{}

func (v RequiredValidator) Validate(val interface{}) (bool, error) {
	switch value := val.(type) {
	case map[string]interface{}:
		if len(value) == 0 {
			return false, fmt.Errorf("the value of required field is not null [%v]", val)
		}
	case []string:
		if len(value) == 0 {
			return false, fmt.Errorf("the value of required field is not null [%v]", val)
		}
	default:
		if reflect.ValueOf(value).IsZero() {
			return false, fmt.Errorf("the value of required field is not null [%v]", val)
		}
	}
	return true, nil

}

// ex: `validate:"requiredInMap: field1, field2,"`
type RequiredInMapValidator struct{ Fields []string }

func (v RequiredInMapValidator) Validate(val interface{}) (bool, error) {
	switch value := val.(type) {
	case map[string]interface{}:
		for _, f := range v.Fields {
			if f == "" {
				break
			}
			_, ok := value[f]
			if !ok {
				return false, fmt.Errorf("required field missing [%v]", f)
			}
		}
		return true, nil
	default:
		return false, fmt.Errorf("cannot check the field exists or not, the type error")
	}
}

// ex: `validate:"requiredSubInMap: key:[field1, field2,field3],"`
type RequiredSubInMapValidator struct{ Fields map[string][]string }

func (v RequiredSubInMapValidator) Validate(val interface{}) (bool, error) {
	switch value := val.(type) {
	case map[string]interface{}:
		for key, tmp := range v.Fields {
			data, ok := value[key]
			if !ok {
				return false, fmt.Errorf("required field missing [%v]", key)
			} else {
				r := RequiredInMapValidator{tmp}
				flag, err := r.Validate(data)
				if !flag {
					return flag, err
				}
			}
		}
	default:
		return false, fmt.Errorf("cannot check the field exists or not, the type error")
	}
	return true, nil
}

func getValidatorFromTag(tag string) (validator Validator) {
	defer func() {
		if err := recover(); err != nil {
			validator = ErrorValidator{Message: "parameter format error"}
		}
	}()
	tag = strings.Replace(tag, " ", "", -1)
	args := strings.Split(tag, ":")
	switch args[0] {
	case "required":
		validator = RequiredValidator{}
	case "requiredInMap":
		validator = RequiredInMapValidator{Fields: strings.Split(args[1], ",")}
	case "requiredSubInMap":
		margs := make(map[string][]string)
		newArgs := strings.Join(args[1:], ":")
		for _, arg := range strings.Split(newArgs, "],") {
			if arg == "" {
				break
			}
			arg = strings.Replace(arg, "[", "", -1)
			argArray := strings.Split(arg, ":")
			margs[argArray[0]] = strings.Split(argArray[1], ",")
		}
		validator = RequiredSubInMapValidator{Fields: margs}
	default:
		validator = DefaultValidator{}
	}
	return
}

// 验证struct嵌套只能验证两层
func ValidateStruct(s interface{}) []error {
	errs := []error{}
	v := reflect.ValueOf(s)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Kind() == reflect.Struct {
			v1 := v.Field(i)
			for j := 0; j < v.NumField(); j++ {
				tag := v1.Type().Field(j).Tag.Get(tagName)
				if tag == "" || tag == "-" {
					continue
				}
				validator := getValidatorFromTag(tag)
				valid, err := validator.Validate(v1.Field(j).Interface())
				if !valid && err != nil {
					errs = append(errs, fmt.Errorf("%s %s", v1.Type().Field(j).Name, err.Error()))
				}
			}
		} else {
			tag := v.Type().Field(i).Tag.Get(tagName)
			if tag == "" || tag == "-" {
				continue
			}
			validator := getValidatorFromTag(tag)
			valid, err := validator.Validate(v.Field(i).Interface())
			if !valid && err != nil {
				errs = append(errs, fmt.Errorf("%s %s", v.Type().Field(i).Name, err.Error()))
			}
		}
	}
	return errs
}
