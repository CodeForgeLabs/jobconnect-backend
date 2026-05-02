package infrastructure

import (
	"job-connect/domain"
	"log"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		domain.User{},
		domain.Job{},
		domain.Milestone{},
	)

	if err != nil {
		log.Printf("Error migrating database: %v\n", err)
		return err
	}

	log.Println("Database migrated successfully")
	return nil
}
