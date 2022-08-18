package client

import (
	"seraph/app/config"
	"seraph/app/objects"
	"seraph/pkg/contextx"
	messaging "seraph/pkg/oslo-messaging-go"
	"seraph/pkg/oslo-messaging-go/message"
	"seraph/pkg/service"
)

type Client struct {
	clt messaging.RPCClient
}

func (c *Client) Initialize() error {
	trans, err := service.GetTransport(config.Config.Messaging.Connection)
	if err != nil {
		return err
	}

	target := message.Target{
		Exchange: config.Config.Messaging.Exchange,
		Topic:    "",
		Host:     "servername",
		Version:  config.Config.Messaging.Version,
	}

	c.clt = messaging.NewRPCClient(trans, target, 0)
	return nil
}

func (c *Client) prepareClient(fanout bool) messaging.CallContext {
	return c.clt.PrepareAdvance(config.Config.Engine.Topic, config.Config.Engine.Host, fanout)
}

func (c *Client) StartWorkflow(ctx *contextx.Context, wfID, wfNamespace, exID string, wfInput map[string]interface{}, description string, params map[string]interface{}) (*objects.WorkflowExecution, error) {
	argv := StartWorkflowArg{
		WorkflowID:          wfID,
		WorkflowNamespace:   wfNamespace,
		WorkflowExecutionID: exID,
		WorkflowInput:       wfInput,
		Description:         description,
		Params:              params,
	}
	callCtx := c.prepareClient(false)

	ex := objects.NewWorkflowExecution()
	err := callCtx.Call(ctx, "start_workflow", argv, 0, ex)
	return ex, err
}

func (c *Client) OnActionComplete(ctx *contextx.Context, actionExID string, result interface{}, wfAction bool) (string, error) {
	argv := OnActionCompleteArg{
		ActionExecutionID: actionExID,
		Result:            result,
		IsWorkflowAction:  wfAction,
	}
	callCtx := c.prepareClient(false)

	exID := ""
	err := callCtx.Call(ctx, "on_action_complete", argv, 0, &exID)
	return exID, err
}

func NewClient() (*Client, error) {
	c := &Client{}
	if err := c.Initialize(); err != nil {
		return nil, err
	}
	return c, nil
}
