package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func NewSqlStorage() (*sql.DB, error) {
	dbType := os.Getenv("DB_TYPE")
	dbUrl := os.Getenv("DB_URL")
	db, err := sql.Open(dbType, dbUrl)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}
	return db, nil
}
