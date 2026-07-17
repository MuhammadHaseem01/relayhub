package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// AppConfig is the global configuration instance.
var AppConfig = IConfig{}

type Config struct {
	Port            string
	DatabaseURL     string
	AuthTokenSecret string
	CORSOrigin      string
}

type IConfig struct {
	// Global settings
	RawVars  map[string]string
	APP_NAME string `env:"APP_NAME" required:"false" default:"cargonex"`

	// Server configs
	SERVER_PORT string `env:"SERVER_PORT" required:"false" default:"5000"`

	// Database settings
	DRIVER_NAME      string `env:"DRIVER_NAME" required:"false" default:"postgres"`
	DATABASE_URL     string `env:"DATABASE_URL" required:"false" default:"postgres://postgres:postgres@localhost:5432/nextroutex?sslmode=disable"`
	DB_HOST          string `env:"DB_HOST" required:"false" default:"localhost"`
	DB_PORT          string `env:"DB_PORT" required:"false" default:"5432"`
	DB_USERNAME      string `env:"DB_USERNAME" required:"false" default:"postgres"`
	DB_PASSWORD      string `env:"DB_PASSWORD" required:"false" default:"postgres"`
	DB_NAME          string `env:"DB_NAME" required:"false" default:"nextroutex"`
	DB_SSLMODE       string `env:"APP_DB_SSLMODE" required:"false" default:"disable"`
	DB_MAX_CONN      int    `env:"DB_MAX_CONN" required:"false" default:"100"`
	DB_MAX_IDLE_CONN int    `env:"DB_MAX_IDLE_CONN" required:"false" default:"10"`

	// Auth and CORS settings
	AUTH_TOKEN_SECRET string `env:"AUTH_TOKEN_SECRET" required:"false" default:"nextroutex-development-secret"`
	CORS_ORIGIN       string `env:"CORS_ORIGIN" required:"false" default:"http://localhost:3000"`
}

func Load() Config {
	return Config{
		Port:            env("PORT", env("SERVER_PORT", "5000")),
		DatabaseURL:     env("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/nextroutex?sslmode=disable"),
		AuthTokenSecret: env("AUTH_TOKEN_SECRET", "nextroutex-development-secret"),
		CORSOrigin:      env("CORS_ORIGIN", "http://localhost:3000"),
	}
}

func env(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func EnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// InitializeConfig dynamically loads environment variables based on struct tags.
func (c *IConfig) InitializeConfig() error {
	c.RawVars = make(map[string]string)

	v := reflect.ValueOf(c).Elem()
	t := reflect.TypeOf(c).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		envTag := field.Tag.Get("env")
		requiredTag := field.Tag.Get("required")
		defaultTag := field.Tag.Get("default")

		if envTag == "" {
			continue // Skip fields without an 'env' tag
		}

		// Fetch environment variable
		value := strings.TrimSpace(os.Getenv(envTag))

		if value == "" && defaultTag != "" {
			value = defaultTag
		}

		// If required and not set, return an error
		if requiredTag == "true" && value == "" {
			return fmt.Errorf("missing required environment variable: %v", envTag)
		}

		// Store in RawVars for reference
		c.RawVars[envTag] = value

		// Dynamically set the field value
		fieldValue := v.Field(i)
		if fieldValue.Kind() == reflect.String {
			fieldValue.SetString(value)
		} else if fieldValue.Kind() == reflect.Int {
			intValue, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("invalid integer value for %v: %v", envTag, err)
			}
			fieldValue.SetInt(int64(intValue))
		} else if fieldValue.Kind() == reflect.Bool {
			boolValue, err := strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("invalid boolean value for %v: %v", envTag, err)
			}
			fieldValue.SetBool(boolValue)
		} else if fieldValue.Kind() == reflect.Float64 {
			floatValue, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("invalid float value for %v: %v", envTag, err)
			}
			fieldValue.SetFloat(floatValue)
		} else if fieldValue.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.String {
			sliceValue := strings.Split(value, ",")
			fieldValue.Set(reflect.ValueOf(sliceValue))
		} else if fieldValue.Type() == reflect.TypeOf(time.Duration(0)) {
			durationValue, err := time.ParseDuration(value)
			if err != nil {
				return fmt.Errorf("invalid duration value for %v: %v", envTag, err)
			}
			fieldValue.Set(reflect.ValueOf(durationValue))
		}
	}

	return nil
}

// InitConfig initializes the global configuration and logs an error if it fails.
func InitConfig() {
	if err := AppConfig.InitializeConfig(); err != nil {
		logrus.Fatalf("Configuration initialization failed: %v", err)
	}
}
