package objects

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"seraph/app/expressions/golang"
	"seraph/app/expressions/jinja"
	"strings"
)

var (
	actionPatterns = map[string]string{
		"command":           `[\w\.]+[^=\(\s\"]*`,
		"golang_expression": golang.AnyRegexp,
		"jinja_expression":  jinja.AnyRegexp,
	}
	itemValuePatterns = []string{
		golang.AnyRegexp,
		jinja.AnyRegexp,
	}
	withItemsPattern = fmt.Sprintf(`\s*([\w\d_\-]+)\s*in\s*(\[.+\]|%s)`, strings.Join(itemValuePatterns, "|"))
	withItemsRegex   = regexp.MustCompile(withItemsPattern)

	EXPRESSION       = `json_parse`
	allInBrackets    = `\[.*\]\s*`
	allInQuotes      = `\"[^\"]*\"\s*`
	allInApostrophes = `'[^']*'\s*`
	_DIGITS          = `\d+`
	_TRUE            = "true"
	_FALSE           = "false"
	_NULL            = "null"

	allSupportElements = []string{
		allInQuotes, allInApostrophes, EXPRESSION,
		allInBrackets, _TRUE, _FALSE, _NULL, _DIGITS,
	}

	paramsPattern = fmt.Sprintf("([-_\\w]+)=(%s)", strings.Join(allSupportElements, "|"))
	paramsRegex   = regexp.MustCompile(paramsPattern)
)

func matchCommand(str string) (string, error) {
	var patternStr []string
	for _, p := range actionPatterns {
		patternStr = append(patternStr, p)
	}

	r, err := regexp.Compile(fmt.Sprintf("^(%s)", strings.Join(patternStr, "|")))
	if err != nil {
		return "", err
	}

	matched := r.FindAllString(str, -1)
	if len(matched) == 0 {
		return "", errors.New("not found any command")
	}

	return matched[0], nil
}

func matchParams(str string) (map[string]interface{}, error) {
	params := map[string]interface{}{}

	for _, match := range paramsRegex.FindAllStringSubmatch(str, -1) {
		name := strings.TrimSpace(match[1])
		value := match[2]
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
			params[name] = value
		} else {
			var v interface{}
			if err := json.Unmarshal([]byte(value), &v); err == nil {
				params[name] = v
			} else {
				params[name] = value
			}
		}
	}

	return params, nil
}

func MatchWithItems(itemsList []string) (map[string]interface{}, error) {
	if itemsList == nil || len(itemsList) == 0 {
		return nil, nil
	}

	items := map[string]interface{}{}

	for _, itemStr := range itemsList {
		for _, match := range withItemsRegex.FindAllStringSubmatch(itemStr, -1) {
			name := strings.TrimSpace(match[1])
			value := match[2]
			if len(value) > 2 && value[0] == '[' {
				var v []interface{}
				if err := json.Unmarshal([]byte(value), &v); err == nil {
					items[name] = v
				} else {
					return nil, errors.New("invalid with-items")
				}
			} else {
				items[name] = value
			}
		}
	}

	return items, nil
}

func ParseCMDAndInputs(cmdStr string) (string, map[string]interface{}, error) {
	cmd, err := matchCommand(cmdStr)
	if err != nil {
		return "", nil, err
	}

	inputs, err := matchParams(cmdStr)
	if err != nil {
		return "", nil, err
	}

	return cmd, inputs, nil
}
