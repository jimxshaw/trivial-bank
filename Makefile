postgres:
	docker run --name postgres12 --network bank-network -p 5432:5432 -e POSTGRES_USER=root -e POSTGRES_PASSWORD=password -d postgres:12-alpine

createdb:
	docker exec -it postgres12 createdb --username=root --owner=root trivial_bank

dropdb:
	docker exec -it postgres12 dropdb trivial_bank

network:
	docker network create bank-network

# make migrate NAME=input_migration_name
migrate:
	migrate create -ext sql -dir db/migration -seq $(NAME)

migrateup:
	migrate -path db/migration -database "postgresql://root:<AWS DB PASSWORD HERE>@<AWS HOST HERE>:5432/trivial_bank" -verbose up 1

migrateupall:
	migrate -path db/migration -database "postgresql://root:password@localhost:5432/trivial_bank?sslmode=disable" -verbose up

migratedown:
	migrate -path db/migration -database "postgresql://root:password@localhost:5432/trivial_bank?sslmode=disable" -verbose down 1

migratedownall:
	migrate -path db/migration -database "postgresql://root:password@localhost:5432/trivial_bank?sslmode=disable" -verbose down

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
