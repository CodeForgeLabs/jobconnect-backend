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
		domain.PortfolioItem{},
		domain.Milestone{},
		domain.Proposal{},
		domain.Contract{},
		domain.ContractMilestone{},
		domain.TimeLog{},
		domain.Message{},
		domain.Conversation{},
		domain.Wallet{},
		domain.WalletTransaction{},
		domain.Review{},
	)

	if err != nil {
		log.Printf("Error migrating database: %v\n", err)
		return err
	}

	log.Println("Database migrated successfully")
	return nil
}
