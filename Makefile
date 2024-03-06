.PHONY: up-container
up-container:
	docker-compose up

.PHONY: build-container
build-container:
	docker-compose build

.PHONY: down-container
down-container:
	docker-compose down

.PHONY: send
send:
	go run ./cmd/sender/

.PHONY: run
run:
	go run ./cmd/service/