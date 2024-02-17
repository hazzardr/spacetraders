PROJECT_NAME := "spacetraders"
EXEC_NAME := spacetraders
SPACE_TRADERS_OPENAPI_URL := "https://stoplight.io/api/v1/projects/spacetraders/spacetraders/nodes/reference/SpaceTraders.json?fromExportButton=true&snapshotType=http_service&deref=optimizedBundle"
POSTGRES_URL ?= "spacetraders:spacetraders@localhost:5432/spacetraders?sslmode=disable"

.PHONY: help ## print this
help:
	@echo ""
	@echo "$(PROJECT_NAME) Development CLI"
	@echo ""
	@echo "Usage:"
	@echo "  make <command>"
	@echo ""
	@echo "Commands:"
	@grep '^.PHONY: ' Makefile | sed 's/.PHONY: //' | awk '{split($$0,a," ## "); printf "  \033[34m%0-10s\033[0m %s\n", a[1], a[2]}'

.PHONY: doctor ## checks if local environment is ready for development
doctor:
	@echo "Checking local environment..."
	@if ! command -v go &> /dev/null; then \
		echo "`go` is not installed. Please install it first."; \
		exit 1; \
	fi
	@if [[ ! ":$$PATH:" == *":$$HOME/go/bin:"* ]]; then \
		echo "GOPATH/bin is not in PATH. Please add it to your PATH variable."; \
		exit 1; \
	fi
	@if ! command -v cobra-cli &> /dev/null; then \
		echo "Cobra-cli is not installed. Please run 'make deps'."; \
		exit 1; \
	fi
	@if ! command -v sqlc &> /dev/null; then \
		echo "`sqlc` is not installed. Please run 'make deps'."; \
		exit 1; \
	fi
	@if ! command -v docker &> /dev/null; then \
		echo "`docker` is not installed. Please install it first."; \
		exit 1; \
	fi
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "`golangci-lint` is not installed. Please install it first."; \
		exit 1; \
	fi
	@if ! command -v migrate &> /dev/null; then \
		echo "`migrate` is not installed. Please run `make deps`."; \
		exit 1; \
	fi
	@echo "Local environment OK"


.PHONY: deps ## install dependencies used for development
deps:
	@echo "Installing dependencies..."
	@go install github.com/spf13/cobra-cli@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
	@echo "Done!"

.PHONY: build ## build the project
build:
	@echo "Building..."
	@go build -gcflags "all=-N -l" -o $(EXEC_NAME)
	@echo "Done!"

.PHONY: run ## run the project
run:
	@echo "Running..."
	@go run main.go serve
	@echo "Done!"

.PHONY: clean ## delete generated code
clean:
	@echo "Deleting generated code..."
	@rm -rf generated
	@echo "Done!"

.PHONY: generate ## generates boilerplate models for db, api, and spacetraders client
generate:
	@echo "Generating database models..."
	@sqlc generate -f sqlc.yaml
	@echo "Generating server code..."
	@mkdir -p generated/api
	@oapi-codegen -generate types,server rest-api.yaml > ./generated/api/server_gen.go
	@echo "Generating spacetraders client..."
	@mkdir -p generated/spacetraders
	@oapi-codegen -generate types,client $(SPACE_TRADERS_OPENAPI_URL) > generated/spacetraders/client_gen.go
	@echo "Done!"

.PHONY: db ## starts postgres database
db:
	@echo "Setting up database..."
	@docker-compose up -d
	@echo "Done!"

.PHONY: migrate ## run database migrations
migrate:
	@echo "Running migrations..."
	@migrate -path ./db/migrations -database "$(POSTGRES_URL)" up
	@echo "Done!"

.PHONY: migrate-down ## tear down database migrations
migrate-down:
	@echo "Running migrations..."
	@migrate -path ./db/migrations -database "postgresql://$(POSTGRES_URL)" down
	@echo "Done!"

.PHONY: create-migration ## create a new db migration. Usage: make create-migration name=<migration_name>
create-migration:
	@echo "Creating migration..."
	@migrate create -ext sql -dir ./db/migrations -seq $(name)
	@echo "Done!"


.PHONY: lint ## run linters
lint:
	@echo "Running linters..."
	@golangci-lint run
	@echo "Done!"

# TODO: test, migrate, seed