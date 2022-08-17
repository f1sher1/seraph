package models

import (
	"seraph/pkg/gormx"
	"time"
)

type Execution struct {
	ID string `gorm:"primaryKey;size:255;"`
	// workflow definition name
	// task definition name
	Name        string          `gorm:"index;size:255"`
	Description string          `gorm:"type:text"`
	Tags        gormx.SliceJson `gorm:"type:mediumtext"`
	// workflow definition id
	WorkflowID string `gorm:"size:255;index"`
	ProjectID  string `gorm:"index;size:255"`
	State      string `gorm:"size:255;index"`
	StateInfo  string `gorm:"type:mediumtext"`

	CreatedAt time.Time `gorm:"default:null"`
	UpdatedAt time.Time `gorm:"default:null"`
	Deleted   int       `gorm:"default:0"`
	DeletedAt time.Time `gorm:"default:null"`

	// 工作流范围，与workflow definition一致
	WorkflowNamespace string `gorm:"size:255;index"`
	// 执行范围，与workflow definition一致
	Scope string `gorm:"index;size:255"`

	// 开始时间
	StartedAt time.Time `gorm:"default:null"`
	// 结束时间
	FinishedAt time.Time `gorm:"default:null"`

	// 第一个workflow execution id
	RootExecutionID string `gorm:"size:255;index"`
}

type WorkflowExecution struct {
	Execution
	// 定义
	Spec string `gorm:"type:mediumtext"`
	// 输入参数
	Input gormx.MapJson `gorm:"type:longtext"`
	// 输出内容
	Output gormx.MapJson `gorm:"type:longtext"`
	// 内部流转参数
	Params gormx.MapJson `gorm:"type:longtext"`

	// 运行时内部参数
	RuntimeContext gormx.MapJson `gorm:"type:longtext"`
	// 用户信息
	Context gormx.MapJson `gorm:"type:longtext"`
	// 工作流名称
	WorkflowName string `gorm:"size:255;index"`

	// 二级工作流时
	TaskExecutionID string `gorm:"size:255;index"`
}

type TaskExecution struct {
	Execution
	// workflow execution id
	WorkflowExecutionID string `gorm:"size:255;index"`

	// task definition
	Spec string `gorm:"type:mediumtext"`
	// 任务发布的变量
	Published gormx.MapJson `gorm:"type:longtext"`

	// 任务类型，workflow或者action
	Type string `gorm:"size:255;index"`
	// 是否已经处理完成
	Processed bool
	// 下级执行的任务列表
	NextTasks gormx.SliceJson `gorm:"type:longtext"`
	// 是否有下级任务
	HasNextTasks bool
	// 异常是否处理
	ErrorHandled bool
	// join时的唯一标识
	UniqueKey string `gorm:"size:255;index"`
	// 备注信息，或者重试信息 {"triggered_by": [{"event": "on-success", "task_id": "6bc5f3dd-beff-46cd-a09c-4f676df81c50"}]}
	RuntimeContext gormx.MapJson `gorm:"type:longtext"`
	// 输入参数
	InContext gormx.MapJson `gorm:"type:longtext"`
	// 工作流名称
	WorkflowName string `gorm:"size:255;index"`
}

type ActionExecution struct {
	Execution
	// action definition
	Spec string `gorm:"type:mediumtext"`
	// 输入参数
	Inputs gormx.MapJson `gorm:"type:longtext"`
	// 输出内容
	Outputs gormx.MapJson `gorm:"type:longtext"`
	// 关联的task execution id
	TaskExecutionID string `gorm:"size:255;index"`
	// 最后一次心跳时间
	LastHeartbeat time.Time `gorm:"default:null"`
	// 备注信息，或者重试信息
	RuntimeContext gormx.MapJson `gorm:"type:longtext"`
	// 工作流名称
	WorkflowName string `gorm:"size:255;index"`
}
