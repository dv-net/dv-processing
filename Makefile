appName     := processing

VERSION ?= $(strip $(shell ./scripts/version.sh))
VERSION_NUMBER := $(strip $(shell ./scripts/version.sh number))
COMMIT_HASH := $(shell git rev-parse --short HEAD)
GO_OPT_BASE := -ldflags "-X main.version=$(VERSION) $(GO_LDFLAGS) -X main.commitHash=$(COMMIT_HASH)"

MIGRATIONS_DIR = ./sql/postgres/migrations/
POSTGRES_DSN = postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_ADDR)/$(POSTGRES_DB_NAME)?sslmode=disable

###################
###    DEV      ###
###################

webhooks:
	go run ./cmd/webhooks start $(filter-out $@,$(MAKECMDGOALS))

.PHONY: start
start:
	go run ./cmd/app start -c config.yaml $(filter-out $@,$(MAKECMDGOALS))

.PHONY: build
build:
	go build $(GO_OPT_BASE) -o bin/$(appName) ./cmd/app

fmt:
	@gofumpt -l -w .

watch:
	@air -c air.toml

lint:
	@golangci-lint run --show-stats

migrate:
	go run ./cmd/app migrate $(filter-out $@,$(MAKECMDGOALS)) --silent

db-drop:
	bin/$(appName) migrate drop -c config.yaml

db-create-migration:
	migrate create -ext sql -dir "$(MIGRATIONS_DIR)" $(filter-out $@,$(MAKECMDGOALS))

grpcui:
	grpcui --plaintext localhost:9000

gensql:
	@pgxgen crud
	@pgxgen sqlc generate

genproto:
	@buf lint
	@find ./api -name "*.pb.go" -type f -delete
	@buf generate

genreadme:
	go run ./cmd/app utils readme

install-dev-tools:
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest && migrate -version
	go install github.com/air-verse/air@latest && air -v
	go install github.com/swaggo/swag/cmd/swag@latest && swag -version
	go install github.com/tkcrm/pgxgen/cmd/pgxgen@latest && pgxgen -version
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && protoc-gen-go --version
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest && protoc-gen-openapiv2 --version
	go install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@latest && protoc-gen-doc --version
	go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest && protoc-gen-connect-go --version
	go install github.com/ethereum/go-ethereum/cmd/abigen@latest && abigen --version

genenvs:
	go run ./cmd/app config genenvs

genabi:
	abigen --abi ./pkg/walletsdk/eth/erc20/erc20.abi --pkg erc20 --type ERC20 --out ./pkg/walletsdk/eth/erc20/erc20.go

gen: gensql genproto genenvs genabi

# Empty goals trap
%:
	@:
