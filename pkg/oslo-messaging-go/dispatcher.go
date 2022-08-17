package messaging

import (
	"errors"
	"fmt"
	"reflect"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"seraph/pkg/oslo-messaging-go/interfaces"
	"unicode"
)

type MethodType struct {
	method  reflect.Method
	ArgType reflect.Type
}

func camelCaseToUnderscore(s string) string {
	var output []rune
	for i, r := range s {
		if i == 0 {
			output = append(output, unicode.ToLower(r))
		} else {
			if unicode.IsUpper(r) {
				output = append(output, '_')
			}

			output = append(output, unicode.ToLower(r))
		}
	}
	return string(output)
}

type Endpoint struct {
	nameMap         map[string]*MethodType
	reflectEndpoint reflect.Value
	serializer      interfaces.Serializer
}

func (e Endpoint) HasMethod(method string) bool {
	_, ok := e.nameMap[method]
	return ok
}

func (e *Endpoint) Call(ctx *contextx.Context, method string, arguments map[string]interface{}) (interface{}, error) {
	if mType, ok := e.nameMap[method]; !ok {
		return nil, fmt.Errorf("method %s not found", method)
	} else {
		argv, err := e.serializer.Deserialize(arguments, mType.ArgType)
		if err != nil {
			return nil, err
		}

		callMethod := e.reflectEndpoint.MethodByName(mType.method.Name)
		results := callMethod.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(argv),
		})

		replyErr := results[1].Interface()
		if replyErr == nil {
			return e.serializer.Serialize(results[0].Interface())
		} else {
			return nil, replyErr.(error)
		}
	}
}

func NewEndpoint(endpointStruct interface{}, serializer interfaces.Serializer) (*Endpoint, error) {
	e := reflect.ValueOf(endpointStruct)
	eType := reflect.TypeOf(endpointStruct)

	if e.Kind() != reflect.Struct && e.Kind() != reflect.Ptr {
		return nil, errors.New("invalid endpoint, must be struct")
	}

	endpoint := &Endpoint{
		nameMap:         map[string]*MethodType{},
		reflectEndpoint: e,
		serializer:      serializer,
	}

	for i := 0; i < eType.NumMethod(); i++ {
		method := eType.Method(i)
		if !method.IsExported() {
			continue
		}

		mType := method.Type
		if mType.NumIn() != 3 {
			log.Debugf(nil, "rpcServer.Register: method %q has %d input parameters; needs exactly three", method.Name, mType.NumIn())
			continue
		}

		callName := camelCaseToUnderscore(method.Name)
		endpoint.nameMap[callName] = &MethodType{
			method:  method,
			ArgType: method.Type.In(2),
		}
	}

	return endpoint, nil
}

type RPCDispatcher struct {
	endpoints  []*Endpoint
	serializer interfaces.Serializer
}

func NewRPCDispatcher() RPCDispatcher {
	return RPCDispatcher{
		endpoints:  []*Endpoint{},
		serializer: &JsonSerializer{},
	}
}

func (r *RPCDispatcher) Dispatch(message interfaces.Message) (interface{}, error) {
	body, err := message.GetBody()
	if err != nil {
		return nil, err
	}

	method := body.GetMethod()
	arguments := body.GetArguments()
	ctx := body.GetContext()

	for _, endpoint := range r.endpoints {
		if endpoint.HasMethod(method) {
			return endpoint.Call(ctx, method, arguments)
		}
	}

	return nil, fmt.Errorf("method %s not found", method)
}

func (r *RPCDispatcher) AddEndpoint(endpoint interface{}) error {
	e, err := NewEndpoint(endpoint, r.serializer)
	if err != nil {
		return err
	}
	r.endpoints = append(r.endpoints, e)
	return nil
}
