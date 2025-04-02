package main

import (
	"github.com/Krchnk/EffectiveMobileFullNameTest/internal/api"
	"github.com/Krchnk/EffectiveMobileFullNameTest/internal/db"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})

	err := godotenv.Load()
	if err != nil {
		logrus.Warn("Error loading .env file")
	}

	database, err := db.InitDB()
	if err != nil {
		logrus.Fatal("Failed to connect to database: ", err)
	}
	defer database.Close()

	if err := db.RunMigrations(database); err != nil {
		logrus.Fatal("Failed to run migrations: ", err)
	}

	serverAddr := db.GetEnv("SERVER_ADDR", ":8080")
	logrus.WithField("addr", serverAddr).Info("Starting server")
	if err := api.StartServer(database, serverAddr); err != nil {
		logrus.Fatal("Server failed: ", err)
	}
}
