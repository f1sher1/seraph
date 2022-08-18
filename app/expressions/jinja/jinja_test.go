package jinja

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

	expr := "{% _.test == 1 %}"
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

	expr := "This is {% _.test == 1 %}, but some one is {% _.test == 2 %}, it's different"
	ok := Match(expr)
	if asserter.Equal(true, ok) {
		result, err := Evaluate(expr, data)
		if asserter.NoError(err) {
			if asserter.Equal("This is True, but some one is False, it's different", result) {
				log.Println(result)
			}
		}
	}
}
