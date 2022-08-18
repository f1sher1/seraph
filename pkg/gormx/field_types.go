package gormx

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

func scan(s interface{}, value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:

		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", v))
	}
}

func value(s interface{}) (driver.Value, error) {
	v := reflect.ValueOf(s)
	if v.IsZero() {
		return nil, nil
	}
	result, err := json.Marshal(s)
	return result, err
}

type SliceJson []interface{}

func (s *SliceJson) Scan(value interface{}) error {
	return scan(&s, value)
}

func (s SliceJson) Value() (driver.Value, error) {
	return value(s)
}

type MapJson map[string]interface{}

func (s *MapJson) Scan(value interface{}) error {
	return scan(&s, value)
}

func (s MapJson) Value() (driver.Value, error) {
	return value(s)
}
