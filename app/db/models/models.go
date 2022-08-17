package models

var (
	Models = []interface{}{
		&WorkflowDefinition{},
		&ActionDefinition{},
		&WorkflowExecution{},
		&ActionExecution{},
		&TaskExecution{},
		&NamedLock{},
	}
)
