package builtin

import (
	"encoding/json"
	"errors"
)

func builtinJSONFunction(values ...interface{}) (interface{}, error) {
	output, err := json.Marshal(values[0])
	if err != nil {
		return nil, errors.New("invalid data")
	}
	return string(output), nil
}
