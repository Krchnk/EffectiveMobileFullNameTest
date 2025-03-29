package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

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
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "secret"),
		getEnv("DB_NAME", "persons_db"),
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
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		logrus.WithError(err).Error("Failed to create migrations table")
		return err
	}

	logrus.Debug("Reading migration files from directory")
	migrationDir := "migrations"
	files, err := os.ReadDir(migrationDir)
	if err != nil {
		logrus.WithError(err).Error("Failed to read migrations directory")
		return err
	}

	var migrationFiles []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".sql" {
			migrationFiles = append(migrationFiles, file.Name())
		}
	}
	sort.Strings(migrationFiles)

	for _, fileName := range migrationFiles {
		logrus.WithField("migration", fileName).Debug("Checking migration status")
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM migrations WHERE name = $1",
			fileName,
		).Scan(&count)
		if err != nil {
			logrus.WithError(err).Error("Failed to check migration status")
			return err
		}

		if count > 0 {
			logrus.WithField("migration", fileName).Debug("Migration already applied, skipping")
			continue
		}

		fullPath := filepath.Join(migrationDir, fileName)
		sqlContent, err := os.ReadFile(fullPath)
		if err != nil {
			logrus.WithField("file", fullPath).WithError(err).Error("Failed to read migration file")
			return err
		}

		logrus.WithField("migration", fileName).Info("Applying migration")
		_, err = db.Exec(string(sqlContent))
		if err != nil {
			logrus.WithField("file", fileName).WithError(err).Error("Failed to apply migration")
			return err
		}

		_, err = db.Exec(
			"INSERT INTO migrations (name) VALUES ($1)",
			fileName,
		)
		if err != nil {
			logrus.WithField("file", fileName).WithError(err).Error("Failed to register migration")
			return err
		}

		logrus.WithField("migration", fileName).Info("Migration applied successfully")
	}
	logrus.Info("Database migrations completed")
	return nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
