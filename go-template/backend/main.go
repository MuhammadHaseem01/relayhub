package main

import (
	"fmt"
	"os"
	"path"
	"runtime"

	"cargonex-backend/src/config"
	"cargonex-backend/src/server"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

// @BasePath /api

// @title Cargonex API
// @version 1.0
// @description Cargonex backend API.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @Security ApiKeyAuth
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host localhost:5000
// @schemes http

func init() {
	// Set the log level to debug
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			// Extract the file name from the full path
			_, file := path.Split(f.File)
			// Return only the file name and line number
			return "", fmt.Sprintf("%s:%d", file, f.Line)
		},
	})
	logrus.SetReportCaller(true) // Enable reporting of the caller's file and line number
}

func main() {

	appEnv := os.Getenv("APP_ENV")
	if appEnv != "prod" {
		envFile := "dev.env"
		_ = godotenv.Load(envFile)
	}

	config.InitConfig()
	cfg := config.Load()

	logrus.Println("Starting Cargonex server")

	if err := server.Start(cfg); err != nil {
		logrus.Fatalf("Error starting server: %v", err)
	}
}
