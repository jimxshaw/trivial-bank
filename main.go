package main

import (
	"database/sql"
	"log"

	"github.com/jimxshaw/trivial-bank/api"
	db "github.com/jimxshaw/trivial-bank/db/sqlc"
	"github.com/jimxshaw/trivial-bank/util"

	_ "github.com/lib/pq"
)

func main() {
	c, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("failed to load configuration:", err)
	}

	conn, err := sql.Open(c.DBDriver, c.DBSource)
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	store := db.NewStore(conn)

	server, err := api.NewServer(store, c)
	if err != nil {
		if err != nil {
			log.Fatal("failed to instantiate new server:", err)
		}
	}

	err = server.Start(c.ServerAddress)
	if err != nil {
		log.Fatal("failed to start server:", err)
	}
}
