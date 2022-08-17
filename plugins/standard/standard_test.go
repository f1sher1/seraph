package standard

import (
	"log"
	"seraph/plugins/plugin"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStdTest_Run(t *testing.T) {
	asserter := assert.New(t)

	input := map[string]interface{}{
		"name": "dinozzo",
		"age":  float64(10),
		"school": map[string]interface{}{
			"junior": "school1",
			"high":   "school2",
		},
	}

	result, err := plugin.Call("std.test", nil, input)

	if asserter.NoError(err) {
		log.Println(result)

		input["replied"] = true
		asserter.Equal(input, result)
	}

}

func TestStdEcho_Run(t *testing.T) {
	asserter := assert.New(t)
	result, err := plugin.Call("std.echo", nil, map[string]string{"output": "test"})

	if asserter.NoError(err) {
		log.Println(result)
		asserter.Equal("test", result)
	}

}
