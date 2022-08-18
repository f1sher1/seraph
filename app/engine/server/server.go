package server

import (
	"seraph/app/config"
	"seraph/app/engine/client"
	"seraph/app/objects"
	"seraph/app/workflow"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	messaging "seraph/pkg/oslo-messaging-go"
	"seraph/pkg/service"
)

func NewEngineServer() *EngineServer {
	return &EngineServer{}
}

type EngineServer struct {
	service.RPCService
	cfg       config.EngineConfig
	rpcServer *messaging.RPCServer
}

func (e *EngineServer) Initialize() error {
	e.cfg = config.Config.Engine
	msgCfg := config.Config.Messaging

	rpcCfg := service.NewRPCServiceConfig(msgCfg.Connection, msgCfg.Exchange, e.cfg.Host, e.cfg.Topic, "2.0", nil)
	rpcServer, err := e.InitializeRPCServer(rpcCfg)
	if err != nil {
		return err
	}

	err = rpcServer.AddEndpoint(&DefaultEngine{})
	if err != nil {
		return err
	}

	e.rpcServer = rpcServer
	return nil
}

func (e *EngineServer) Start() error {
	return e.rpcServer.Start()
}

func (e *EngineServer) Stop() error {
	return e.rpcServer.Stop()
}

type DefaultEngine struct {
}

func (e *DefaultEngine) OnActionComplete(ctx *contextx.Context, argv client.OnActionCompleteArg) (string, error) {
	log.Debugf(ctx, "Default Engine on actioncomlete argv %v", argv)

	return argv.ActionExecutionID, objects.Transaction(ctx, func(subCtx *contextx.Context) error {
		if argv.IsWorkflowAction {
			wfEx, err := objects.QueryWorkflowExecutionByID(subCtx, argv.ActionExecutionID)
			if err != nil {
				return err
			}
			if argv.Result == nil {
				argv.Result = &objects.ActionResult{
					Data: wfEx.Output,
				}
			}

			workflow.OnWorkflowActionComplete(wfEx, argv.GetResult())
		} else {
			actEx, err := objects.QueryActionExecutionByID(subCtx, argv.ActionExecutionID)
			if err != nil {
				return err
			}
			workflow.OnActionComplete(actEx, argv.GetResult())
		}
		return nil
	})
}

func (e *DefaultEngine) StartWorkflow(ctx *contextx.Context, argv client.StartWorkflowArg) (*objects.WorkflowExecution, error) {
	log.Debugf(ctx, "Default Engine start workflow argv %v", argv)

	//wf, err := objects.QueryWorkflowDefinitionByID(ctx, argv.WorkflowID)

	ex := objects.NewWorkflowExecution()
	ex.ID = argv.WorkflowExecutionID
	ex.WorkflowID = argv.WorkflowID
	ex.WorkflowNamespace = argv.WorkflowNamespace
	ex.Description = argv.Description
	ex.Input = argv.WorkflowInput
	return ex, nil
}
