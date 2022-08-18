package db

import (
	"seraph/app/db/models"

	"gorm.io/gorm"
)

func Migrate() error {
	return dbConn.Transaction(func(tx *gorm.DB) error {
		for _, modObj := range models.Models {
			if err := tx.AutoMigrate(modObj); err != nil {
				return err
			}
		}
		return nil
	})
}
