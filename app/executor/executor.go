package executor

import (
	"encoding/json"
	"fmt"
	"seraph/app/engine/client"
	"seraph/app/objects"
	"seraph/pkg/contextx"
	"seraph/pkg/log"
	"seraph/plugins/plugin"
	"time"
)

type Executor interface {
	Initialize() error
	RunAction(ctx *contextx.Context, actionEx *objects.ActionExecution, actionExID string, name string, attributes objects.Table, params objects.Table, execCtx objects.Table, target string, timeout time.Duration) error
}

type BaseExecutor struct {
	EngineClient *client.Client
}

func (e *BaseExecutor) Initialize() error {
	c, err := client.NewClient()
	if err != nil {
		return err
	}
	e.EngineClient = c
	return nil
}

type LocalExecutor struct {
	BaseExecutor
}

func (e *LocalExecutor) callPlugin(name string, attributes objects.Table, params objects.Table) (interface{}, error) {
	return plugin.Call(name, attributes, params)
}

func (e *LocalExecutor) callPluginWithTimeout(name string, attributes objects.Table, params objects.Table, timeout time.Duration) (interface{}, error) {
	resultChan := make(chan interface{}, 1)
	errChan := make(chan error, 1)
	go func(resChan chan interface{}, eChan chan error) {
		result, err := e.callPlugin(name, attributes, params)
		if err != nil {
			eChan <- err
		} else {
			resChan <- result
		}
	}(resultChan, errChan)

	var err error
	var result interface{}
	select {
	case result = <-resultChan:
	case err = <-errChan:
	case <-time.After(timeout):
		err = fmt.Errorf("run action %s timeout after %s", name, timeout)
	}
	return result, err
}

func (e *LocalExecutor) RunAction(ctx *contextx.Context, actionEx *objects.ActionExecution, actionExID string, name string, attributes objects.Table, params objects.Table, execCtx objects.Table, target string, timeout time.Duration) error {
	var err error
	var result interface{}

	// BUG: connot find ActionExecution
	// actEx, err := objects.QueryActionExecutionByID(ctx, actionExID)
	// if err != nil {
	// 	return err
	// }

	attributes = attributes.MergeToNew()
	attributes["execution_context"] = execCtx
	attributes["context"] = ctx.GetMap()
	if timeout > 0 {
		result, err = e.callPluginWithTimeout(name, attributes, params, timeout)
	} else {
		result, err = e.callPlugin(name, attributes, params)
	}

	if err != nil {
		log.Errorf(ctx, "call plugin error %v", err)
		return err
	}
	// map --->  struct
	b, err := json.Marshal(result)
	if err != nil {
		return err
	}
	var ob_result objects.ActionResult
	if err := json.Unmarshal(b, &ob_result); err != nil {
		return err
	}
	log.Debugf(ctx, "Run RPC function return result %#v  --->  %#v", result, ob_result)

	_, err = e.EngineClient.OnActionComplete(ctx, actionExID, &ob_result, actionEx.WorkflowName != "")
	if err != nil {
		// 通知失败不认为任务执行失败了
		log.Errorf(ctx, "Notify engine failed, error: %s", err.Error())
	}
	return nil
}

type RemoteExecutor struct {
	BaseExecutor
}

func (e RemoteExecutor) RunAction(ctx *contextx.Context, actionEx *objects.ActionExecution, actionExID string, name string, attributes objects.Table, params objects.Table, execCtx objects.Table, target string, timeout time.Duration) error {
	panic("implement me")
}

func GetExecutor(kind string) Executor {
	var worker Executor
	switch kind {
	case "local":
		worker = &LocalExecutor{}
	case "remote":
		worker = &RemoteExecutor{}
	default:
		panic("invalid executor type " + kind)
	}
	err := worker.Initialize()
	if err != nil {
		panic(err.Error())
	}
	return worker
}
