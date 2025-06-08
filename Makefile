proto:
	protoc \
	--go_out=. \
	--go-grpc_out=. \
	proto/services.proto

build:
	docker compose build

up:
	docker compose up --build

down:
	docker compose down --volumes

.PHONY: proto build up down down