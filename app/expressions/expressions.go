package expressions

import (
	"reflect"
	"seraph/app/expressions/golang"
	"seraph/app/expressions/jinja"
	"seraph/pkg/log"
)

type Expression interface {
	Match(expr string) bool
	Evaluate(expr string, data map[string]interface{}) (interface{}, error)
}

var (
	builtinExpressions = []Expression{
		golang.GolangExpression{},
		jinja.JinjaExpression{},
	}
)

func Evaluate(expr string, dataCtx map[string]interface{}) (interface{}, error) {
	for _, expression := range builtinExpressions {
		if expression.Match(expr) {
			return expression.Evaluate(expr, dataCtx)
		}
	}
	// 直接是key
	result, ok := dataCtx[expr]
	if ok {
		return result, nil
	}
	return expr, nil
	//return nil, fmt.Errorf("unknown expression '%s'", expr)
}

func EvaluateRecursively(data interface{}, dataCtx map[string]interface{}) (interface{}, error) {
	switch reflect.ValueOf(data).Kind() {
	case reflect.Slice:
		var result []interface{}

		for _, one := range data.([]interface{}) {
			r, err := EvaluateRecursively(one, dataCtx)
			if err != nil {
				return nil, err
			}
			result = append(result, r)
		}

		return result, nil

	case reflect.String:
		r, err := Evaluate(data.(string), dataCtx)
		if err != nil {
			log.Debugf(nil, "Expression %s is not evaluated, [context=%#v]: %s", data.(string), dataCtx, err.Error())
			return data, nil
		}
		return r, nil

	case reflect.Map:
		result := map[string]interface{}{}

		for k, v := range data.(map[string]interface{}) {
			r, err := EvaluateRecursively(v, dataCtx)
			if err != nil {
				return nil, err
			}
			result[k] = r
		}

		return result, nil

	default:
		return data, nil
	}
}
