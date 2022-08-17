package messaging

import (
	"encoding/json"
	"reflect"
)

type JsonSerializer struct {

}

func (s JsonSerializer) Serialize(data interface{}) (interface{}, error) {
	switch reflect.ValueOf(data).Kind() {
	case reflect.Struct:
		// 生成map
		var _data map[string]interface{}
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(jsonBytes, &_data)
		if err != nil {
			return nil, err
		}
		return _data, nil
	default:
		return data, nil
	}
}

func (s JsonSerializer) Deserialize(data interface{}, argType reflect.Type) (interface{}, error) {
	var argv reflect.Value
	// Decode the argument value.
	argIsValue := false // if true, need to indirect before calling.
	if argType.Kind() == reflect.Ptr {
		argv = reflect.New(argType.Elem())
	} else {
		argv = reflect.New(argType)
		argIsValue = true
	}

	switch argType.Kind() {
	case reflect.Struct:
		_data, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(_data, argv.Interface())
		if err != nil {
			return nil, err
		}

		if argIsValue {
			argv = argv.Elem()
		}
	default:
		return data, nil
	}
	return argv.Interface(), nil
}

