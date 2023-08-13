createdb:
	docker exec -it postgres12 createdb --username=root --owner=root trivial_bank

dropdb:
	docker exec -it postgres12 dropdb trivial_bank

.PHONY: createdb