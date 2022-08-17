package workflow

import (
	"seraph/app/objects"
	"seraph/app/workflow/interfaces"
	"seraph/app/workflow/states"
)

var (
	backlogCommandsName = "backlog_commands"
)

func SaveWorkflowCommandToBacklog(wfEx *objects.WorkflowExecution, task interfaces.Task) {
	var backlogCommands []interface{}
	_, ok := wfEx.RuntimeContext[backlogCommandsName]
	if !ok {
		wfEx.RuntimeContext[backlogCommandsName] = backlogCommands
	}

	backlogCommands = append(backlogCommands, task.ToMap())
	wfEx.RuntimeContext[backlogCommandsName] = backlogCommands
}

func DispatchWorkflowTasks(wfExec *objects.WorkflowExecution, tasks []interfaces.Task) error {
	// todo;
	// run command from backlog

	// run new command
	for _, task := range tasks {
		if states.IsCompleted(wfExec.State) {
			break
		}

		if states.IsPaused(wfExec.State) {
			SaveWorkflowCommandToBacklog(wfExec, task)
			break
		}

		if err := task.Run(); err != nil {
			task.ForceFail(err)
			continue
		} else {
			// 触发下级任务
			CheckAffectedTasks(task)
		}
	}

	return nil
}
