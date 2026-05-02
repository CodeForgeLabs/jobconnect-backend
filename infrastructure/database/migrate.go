// package infrastructure

// import (
// 	"log"

// 	"gorm.io/gorm"
// )

// func Migrate(db *gorm.DB) error {
// 	err := db.AutoMigrate(
// 		entities.User{},
// 		entities.Message{},
// 		entities.Conversation{},
// 	)

// 	if err != nil {
// 		log.Printf("Error migrating database: %v\n", err)
// 		return err
// 	}

// 	log.Println("Database migrated successfully")
// 	return nil
// }
