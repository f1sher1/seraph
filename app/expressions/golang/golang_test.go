package golang

import (
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestEvaluateBool(t *testing.T) {
	asserter := assert.New(t)
	data := map[string]interface{}{
		"test": 1,
	}

	expr := "{{eq .test 1}}"
	ok := Match(expr)
	if asserter.Equal(true, ok) {
		result, err := Evaluate(expr, data)
		if asserter.NoError(err) {
			if asserter.Equal(true, result) {
				log.Println(result)
			}
		}
	}
}

func TestEvaluateString(t *testing.T) {
	asserter := assert.New(t)
	data := map[string]interface{}{
		"test": 1,
	}

	expr := "This is {{ eq .test 1 }}, but some one is {{ eq .test 2 }}, it's different"
	ok := Match(expr)
	if asserter.Equal(true, ok) {
		result, err := Evaluate(expr, data)
		if asserter.NoError(err) {
			if asserter.Equal("This is true, but some one is false, it's different", result) {
				log.Println(result)
			}
		}
	}
}
