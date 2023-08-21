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
var testDB *sql.DB

func TestMain(m *testing.M) {
	var err error

	testDB, mock, err = sqlmock.New()
	if err != nil {
		log.Fatalf("failed to connect to mock database connection: %v", err)
	}
	defer testDB.Close()

	testQueries = New(testDB)

	os.Exit(m.Run())
}
