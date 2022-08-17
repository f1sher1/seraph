package messaging

import (
	"errors"
	"log"
	"seraph/pkg/contextx"
	"testing"

	"github.com/stretchr/testify/assert"
)

var TestRPCServerEndpointNameMap = map[string]string{
	"public_method":                    "PublicMethod",
	"public_error_method":              "PublicErrorMethod",
	"public_method_with_data_returned": "PublicMethodWithDataReturned",
}

type TestEndpointArgs struct {
	A string                 `json:"a"`
	B map[string]interface{} `json:"b"`
}

type TestRPCServerEndpoint struct {
}

func (t TestRPCServerEndpoint) privateMethod(ctx *contextx.Context, data map[string]interface{}) (interface{}, error) {
	log.Printf("private method %v, data %v", ctx, data)
	return "this is private method", nil
}

func (t TestRPCServerEndpoint) PublicMethod(ctx *contextx.Context, data map[string]interface{}) (interface{}, error) {
	log.Printf("public method %v, data %v", ctx, data)
	return "this is public method", nil
}

func (t TestRPCServerEndpoint) PublicErrorMethod(ctx *contextx.Context, data map[string]interface{}) (interface{}, error) {
	log.Printf("public err method %v, data %v", ctx, data)
	return "this is public error method", errors.New("this is a error")
}

func (t TestRPCServerEndpoint) PublicMethodWithDataReturned(ctx *contextx.Context, data map[string]interface{}) (interface{}, error) {
	log.Printf("public method with data returned %v, data %v", ctx, data)
	data["server"] = "this is public method"
	return data, nil
}

func (t TestRPCServerEndpoint) PublicMethodWithStructReturned(ctx *contextx.Context, data TestEndpointArgs) (interface{}, error) {
	log.Printf("public method with struct returned %v, data %v", ctx, data)
	data.A = "this is dest struct"
	data.B["aa"] = "this is map struct"
	return data, nil
}

func TestRPCDispatcher_AddEndpoint(t *testing.T) {
	asserter := assert.New(t)

	dispatcher := NewRPCDispatcher()
	err := dispatcher.AddEndpoint(TestRPCServerEndpoint{})
	if !asserter.NoError(err) {
		return
	}

	if asserter.Equal(1, len(dispatcher.endpoints)) {
		if asserter.Equal(TestRPCServerEndpointNameMap, dispatcher.endpoints[0].nameMap) {
			log.Println(dispatcher.endpoints[0].nameMap)
		}
	}

}
