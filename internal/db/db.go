package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	migrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

func InitDB() (*sql.DB, error) {
	err := godotenv.Load()
	if err != nil {
		logrus.Warn("Error loading .env file, using system environment variables")
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		GetEnv("DB_HOST", "localhost"),
		GetEnv("DB_PORT", "5432"),
		GetEnv("DB_USER", "postgres"),
		GetEnv("DB_PASSWORD", "secret"),
		GetEnv("DB_NAME", "persons_db"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		logrus.WithError(err).Error("Failed to open database connection")
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		logrus.WithError(err).Error("Failed to ping database")
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	logrus.Info("Successfully connected to database")
	return db, nil
}

func RunMigrations(db *sql.DB) error {
	logrus.Info("Starting database migrations")

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		logrus.WithError(err).Error("Failed to create migration driver")
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		logrus.WithError(err).Error("Failed to initialize migrations")
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		logrus.WithError(err).Error("Failed to apply migrations")
		return err
	}

	logrus.Info("Database migrations completed")
	return nil
}

func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
