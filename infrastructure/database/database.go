package infrastructure

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	TimeZone string
}

func LoadDatabaseConfig() DatabaseConfig {

	return DatabaseConfig{
		Host:     GetEnv("DB_HOST", "localhost"),
		Port:     GetEnv("DB_PORT", "5432"),
		User:     GetEnv("DB_USER", "postgres"),
		Password: GetEnv("DB_PASSWORD", ""),
		DBName:   GetEnv("DB_NAME", "jobconnect-app"),
		SSLMode:  GetEnv("DB_SSL_MODE", "disable"),
		TimeZone: GetEnv("DB_TIMEZONE", "UTC"),
	}
}

func NewDatabase() (*gorm.DB, error) {
	config := LoadDatabaseConfig()
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s", config.Host, config.User, config.Password, config.DBName, config.Port, config.SSLMode, config.TimeZone)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)

	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
		return nil, err
	}

	return db, nil
}

func GetEnv(key, defaultValue string) string {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}
