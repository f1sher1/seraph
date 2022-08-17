package jinja

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"seraph/app/expressions/builtin"
	"strings"

	"github.com/flosch/pongo2/v4"
)

var (
	AnyRegexp   = `\{\%.*\%\}`
	JinjaRegexp = `\{\%(.*?)\%\}`

	reIdentifier = regexp.MustCompile(AnyRegexp)
	reExpression = regexp.MustCompile(JinjaRegexp)
)

type JinjaExpression struct {
}

func (e JinjaExpression) Match(expr string) bool {
	return Match(expr)
}

func (e JinjaExpression) Evaluate(expr string, data map[string]interface{}) (interface{}, error) {
	return Evaluate(expr, data)
}

func Match(expr string) bool {
	return reIdentifier.MatchString(expr)
}

func EvaluateReturnInterface(expr string, data map[string]interface{}) (interface{}, error) {
	tpl, err := pongo2.FromString(expr)
	if err != nil {
		return nil, err
	}

	ctx := pongo2.Context{}
	ctx["_"] = data
	for k, v := range builtin.BuiltinFunc {
		ctx[k] = v
	}

	result, err := tpl.Execute(ctx)
	if err != nil {
		return nil, err
	}

	if result != "" {
		var published interface{}
		err = json.Unmarshal([]byte(result), &published)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("%s, output: '%s'", err.Error(), result))
		}

		return published, nil
	} else {
		return nil, nil
	}
}

func EvaluateReturnString(expr string, data map[string]interface{}) (interface{}, error) {
	tpl, err := pongo2.FromString(expr)
	if err != nil {
		return nil, err
	}

	ctx := pongo2.Context{}
	ctx["_"] = data
	for k, v := range builtin.BuiltinFunc {
		ctx[k] = v
	}

	return tpl.Execute(ctx)
}

func Evaluate(expr string, data map[string]interface{}) (interface{}, error) {
	matched := reExpression.FindAllStringSubmatchIndex(expr, -1)

	if len(matched) == 1 && matched[0][0] == 0 && matched[0][1] == len(expr)-1 {
		// 只有表达式，需要返回实际的内容
		tplStr := fmt.Sprintf(`{{ json(%s) }}`, expr[matched[0][2]:matched[0][3]])
		return EvaluateReturnInterface(tplStr, data)
	} else {
		// 当模板解析，返回字符串
		exprParts := []string{}
		lastPos := 0
		for i := 0; i < len(matched); i++ {
			values := matched[i]
			exprPart := fmt.Sprintf(`{{ %s }}`, expr[values[2]:values[3]])

			exprParts = append(exprParts, expr[lastPos:values[0]])
			exprParts = append(exprParts, exprPart)

			if i == len(matched)-1 {
				// 最后一个
				exprParts = append(exprParts, expr[values[1]:])
			} else {
				lastPos = values[1]
			}
		}

		tplStr := strings.Join(exprParts, "")
		return EvaluateReturnString(tplStr, data)
	}
}
