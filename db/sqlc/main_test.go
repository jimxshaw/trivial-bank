package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/lib/pq"
)

var testQueries *Queries
var mock sqlmock.Sqlmock

func TestMain(m *testing.M) {
	var db *sql.DB
	var err error

	db, mock, err = sqlmock.New()
	if err != nil {
		log.Fatalf("failed to connect to mock database connect: %v", err)
	}
	defer db.Close()

	testQueries = New(db)

	os.Exit(m.Run())
}
