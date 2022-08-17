package messaging

import (
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"seraph/pkg/oslo-messaging-go/driver"
	"seraph/pkg/oslo-messaging-go/message"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRPCServer_ProcessIncoming(t *testing.T) {
	asserter := assert.New(t)

	dialector, err := driver.Open("rabbit://guest:guest@0.0.0.0:5672/")
	if !asserter.NoError(err) {
		return
	}

	transport, err := NewTransport(dialector)
	if !asserter.NoError(err) {
		return
	}

	target := message.Target{
		Exchange: "test-exchange",
		Topic:    "test-topic",
		Host:     "servername",
		Version:  "2.0",
	}

	dispatcher := NewRPCDispatcher()
	err = dispatcher.AddEndpoint(TestRPCServerEndpoint{})
	if !asserter.NoError(err) {
		return
	}

	server := NewRPCServer(transport, target, &dispatcher)
	err = server.Start()
	if !asserter.NoError(err) {
		return
	}

	log.Debugf(nil, "rpc server started")
	for {
		select {
		case <-time.Tick(5 * time.Second):
			continue
		}
	}
}

func TestRPCClient_PrepareAndCall(t *testing.T) {
	asserter := assert.New(t)

	dialector, err := driver.Open("rabbit://guest:guest@0.0.0.0:5672/")
	if !asserter.NoError(err) {
		return
	}

	transport, err := NewTransport(dialector)
	if !asserter.NoError(err) {
		return
	}

	target := message.Target{
		Exchange: "test-exchange",
		Topic:    "test-topic",
		Host:     "servername",
		Version:  "2.0",
	}

	client := NewRPCClient(transport, target, 0)
	callCtx := client.Prepare()

	ctx := contextx.NewContext()
	ctx.Set("project_id", "test-project")

	result := ""
	err = callCtx.Call(ctx, "public_method", map[string]interface{}{"key": "value"}, 0, &result)
	if !asserter.NoError(err) {
		return
	}

	if asserter.Equal("this is public method", result) {

	}
}

func TestRPCClient_PrepareAndCallWithError(t *testing.T) {
	asserter := assert.New(t)

	dialector, err := driver.Open("rabbit://guest:guest@0.0.0.0:5672/")
	if !asserter.NoError(err) {
		return
	}

	transport, err := NewTransport(dialector)
	if !asserter.NoError(err) {
		return
	}

	target := message.Target{
		Exchange: "test-exchange",
		Topic:    "test-topic",
		Host:     "servername",
		Version:  "2.0",
	}

	client := NewRPCClient(transport, target, 0)
	callCtx := client.Prepare()

	ctx := contextx.NewContext()
	ctx.Set("project_id", "test-project")

	result := ""

	err = callCtx.Call(ctx, "public_error_method", map[string]interface{}{"key": "value"}, 0, &result)
	if asserter.Error(err) {
		if asserter.Equal("this is a error", err.Error()) {
			asserter.Equal(nil, result)
		}
	}
}

func TestRPCClient_PrepareAndCallDataReturned(t *testing.T) {
	asserter := assert.New(t)

	dialector, err := driver.Open("rabbit://guest:guest@0.0.0.0:5672/")
	if !asserter.NoError(err) {
		return
	}

	transport, err := NewTransport(dialector)
	if !asserter.NoError(err) {
		return
	}

	target := message.Target{
		Exchange: "test-exchange",
		Topic:    "test-topic",
		Host:     "servername",
		Version:  "2.0",
	}

	client := NewRPCClient(transport, target, 0)

	ctx := contextx.NewContext()
	ctx.Set("project_id", "test-project")

	callCtx := client.Prepare()
	data := map[string]interface{}{"key": "value"}

	var result = map[string]interface{}{}
	err = callCtx.Call(ctx, "public_method_with_data_returned", data, 0, &result)
	if !asserter.NoError(err) {
		return
	}

	data["server"] = "this is public method"
	if asserter.Equal(data, result) {

	}
}

func TestRPCClient_PrepareAndCallStructReturned(t *testing.T) {
	asserter := assert.New(t)

	dialector, err := driver.Open("rabbit://guest:guest@0.0.0.0:5672/")
	if !asserter.NoError(err) {
		return
	}

	transport, err := NewTransport(dialector)
	if !asserter.NoError(err) {
		return
	}

	target := message.Target{
		Exchange: "test-exchange",
		Topic:    "test-topic",
		Host:     "servername",
		Version:  "2.0",
	}

	client := NewRPCClient(transport, target, 0)

	ctx := contextx.NewContext()
	ctx.Set("project_id", "test-project")

	callCtx := client.Prepare()
	data := TestEndpointArgs{
		A: "this is from struct",
		B: map[string]interface{}{
			"cc": "this from map struct",
		},
	}

	var result = TestEndpointArgs{}
	err = callCtx.Call(ctx, "public_method_with_struct_returned", data, 0, &result)
	if !asserter.NoError(err) {
		return
	}

	exceptedResult := TestEndpointArgs{
		A: "this is dest struct",
		B: map[string]interface{}{
			"aa": "this is map struct",
			"cc": "this from map struct",
		},
	}
	if asserter.Equal(exceptedResult, result) {

	}
}

func TestRPCClient_PrepareAndCallMulti(t *testing.T) {
	asserter := assert.New(t)

	dialector, err := driver.Open("rabbit://guest:guest@0.0.0.0:5672/")
	if !asserter.NoError(err) {
		return
	}

	transport, err := NewTransport(dialector)
	if !asserter.NoError(err) {
		return
	}

	target := message.Target{
		Exchange: "test-exchange",
		Topic:    "test-topic",
		Host:     "servername",
		Version:  "2.0",
	}

	f := func(i int, w *sync.WaitGroup) {
		client := NewRPCClient(transport, target, 0)

		ctx := contextx.NewContext()
		ctx.Set("project_id", "test-project")

		callCtx := client.Prepare()
		data := map[string]interface{}{"key" + strconv.Itoa(i): time.Now().String()}

		var result = map[string]interface{}{}
		err = callCtx.Call(ctx, "public_method_with_data_returned", data, 0, &result)

		if !asserter.NoError(err) {
			return
		}

		data["server"] = "this is public method"
		if asserter.Equal(data, result) {

		}
	}

	g := &sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		g.Add(1)
		go f(i, g)
	}

	g.Wait()
}
