package models

import (
	"seraph/pkg/gormx"
	"time"
)

type Definition struct {
	ID          string          `gorm:"primaryKey;size:255;"`
	Name        string          `gorm:"index;size:255"`
	Description string          `gorm:"type:text"`
	Definition  string          `gorm:"type:mediumtext"`
	Tags        gormx.SliceJson `gorm:"type:mediumtext"`
	Spec        string          `gorm:"type:mediumtext"`
	Scope       string          `gorm:"index;size:255"`
	ProjectID   string          `gorm:"index;size:255"`

	CreatedAt time.Time `gorm:"default:null"`
	UpdatedAt time.Time `gorm:"default:null"`
	Deleted   int       `gorm:"default:0"`
	DeletedAt time.Time `gorm:"default:null"`
}

type WorkflowDefinition struct {
	Definition
	Namespace string `gorm:"index;size:255"`
}

type ActionDefinition struct {
	Definition
	Inputs string `gorm:"type:longtext"`

	ActionClass string        `gorm:"size:255"`
	Attributes  gormx.MapJson `gorm:"type:mediumtext"`
}
