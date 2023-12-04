DB_URL=postgresql://root:password@localhost:5432/trivial_bank?sslmode=disable

postgres:
	docker run --name postgres12 --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=password -d postgres:12-alpine

createdb:
	docker exec -it postgres12 createdb --username=root --owner=root trivial_bank

dropdb:
	docker exec -it postgres12 dropdb trivial_bank

# npm dbdocs package
db_docs:
	dbdocs build doc/db.dbml

#npm dbml/cli pckage
db_schema:
	dbml2sql --postgres -o doc/schema.sql doc/db.dbml

network:
	docker network create bank-network

# make migrate NAME=input_migration_name
migrate:
	migrate create -ext sql -dir db/migration -seq $(NAME)

# "postgresql://root:<AWS DB PASSWORD HERE>@<AWS HOST HERE>:5432/trivial_bank" 
migrateup:
	migrate -path db/migration -database "$(DB_URL)" -verbose up 1

migrateupall:
	migrate -path db/migration -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path db/migration -database "$(DB_URL)" -verbose down 1

migratedownall:
	migrate -path db/migration -database "$(DB_URL)" -verbose down

sqlc:
	sqlc generate

test:
	# Testing all packages within the project.
	go test -v -cover ./... 

server:
	go run main.go

mock:
	mockgen -package=mockdb -destination=db/mocks/store.go github.com/jimxshaw/trivial-bank/db/sqlc Store

.PHONY: 
	postgres
	createdb
	dropdb
	db_docs
	db_schema
	network
	migrate
	migrateup
	migrateupall
	migratedown
	migratedownall
	sqlc
	test
	server
	mock
