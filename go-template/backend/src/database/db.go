package database

import (
	"database/sql"
	"time"

	"cargonex-backend/src/config"
	_ "github.com/lib/pq"
)

func New(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(config.EnvInt("DB_MAX_OPEN_CONNS", 25))
	db.SetMaxIdleConns(config.EnvInt("DB_MAX_IDLE_CONNS", 25))
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
