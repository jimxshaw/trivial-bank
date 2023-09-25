package main

import (
	"database/sql"
	"log"

	"github.com/jimxshaw/trivial-bank/api"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"

	_ "github.com/lib/pq"
)

// TODO: Refactor constants to load from Environment Variables.
const (
	dbDriver      = "postgres"
	dbSource      = "postgresql://root:password@localhost:5432/trivial_bank?sslmode=disable"
	serverAddress = "0.0.0.0:8080"
)

func main() {
	conn, err := sql.Open(dbDriver, dbSource)
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	store := db.NewStore(conn)
	server := api.NewServer(store)

	err = server.Start(serverAddress)
	if err != nil {
		log.Fatal("failed to start server:", err)
	}
}
