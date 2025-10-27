package database

import (
	"fmt"
	"log"

	"github.com/yukikurage/task-management-api/internal/config"
	"github.com/yukikurage/task-management-api/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect(cfg *config.Config) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Println("Database connection established")
	return nil
}

func Migrate() error {
	log.Println("Running database migrations...")
	err := DB.AutoMigrate(
		&models.User{},
		&models.Organization{},
		&models.OrganizationMember{},
		&models.Task{},
		&models.TaskAssignment{},
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Println("Database migrations completed")
	return nil
}

func GetDB() *gorm.DB {
	return DB
}

// SetDB sets the database instance (used for testing)
func SetDB(db *gorm.DB) {
	DB = db
}
