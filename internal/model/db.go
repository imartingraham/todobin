package model

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	// pg sets up postgres sql
	_ "github.com/lib/pq"
	"github.com/imartingraham/todobin/internal/util"
)

var db *sql.DB

func init() {
	dbHost := os.Getenv("POSTGRES_HOST")
	dbName := os.Getenv("POSTGRES_DB")
	dbUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("POSTGRES_PASSWORD")

	var err error

	dataSourceName := fmt.Sprintf("user=%s password=%s dbname=%s host=%s sslmode=disable", dbUser, dbPassword, dbName, dbHost)
	db, err = sql.Open("postgres", dataSourceName)
	if err != nil {
		util.Airbrake.Notify(fmt.Errorf("Failed to open DB connection: %w", err), nil)
		log.Panic(err)
	}

	if err = db.Ping(); err != nil {
		util.Airbrake.Notify(fmt.Errorf("Failed to Ping DB: %w", err), nil)
		log.Panic(err)
	}
}
