package models

import "time"

type NamedLock struct {
	ID        string    `gorm:"primaryKey;size:255;"`
	Name      string    `gorm:"index;size:255;unique"`
	CreatedAt time.Time `gorm:"default:null"`
	UpdatedAt time.Time `gorm:"default:null"`
}
