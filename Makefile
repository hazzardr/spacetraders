PROJECT_NAME := "spacetraders"
EXEC_NAME := spacetraders
SPACE_TRADERS_OPENAPI_URL := "https://stoplight.io/api/v1/projects/spacetraders/spacetraders/nodes/reference/SpaceTraders.json?fromExportButton=true&snapshotType=http_service&deref=optimizedBundle"

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
	@echo "Local environment OK"


.PHONY: deps ## install dependencies used for development
deps:
	@echo "Installing dependencies..."
	@go install github.com/spf13/cobra-cli@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2
	@echo "Done!"

.PHONY: build ## build the project
build:
	@echo "Building..."
	@go build -o $(EXEC_NAME) ./main.go
	@echo "Done!"

.PHONY: clean ## delete generated code
clean:
	@echo "Deleting generated code..."
	@rm -rf generated
	@echo "Done!"

.PHONY: generate ## generates boilerplate
generate:
	$(MAKE) generate-server
	$(MAKE) generate-client

.PHONY: generate-server ## generate server and database code
generate-server:
	@echo "Generating database models..."
	@sqlc generate -f sqlc.yaml
	@echo "Generating server code..."
	@mkdir -p generated/api
	@oapi-codegen -generate types,chi-server rest-api.yaml > ./generated/api/server_gen.go
	@echo "Done!"

.PHONY: generate-client ## generate go code based on spacetrader openapi
generate-client:
	@echo "Generating spacetraders client..."
	@mkdir -p generated/spacetraders
	@oapi-codegen -generate types,client $(SPACE_TRADERS_OPENAPI_URL) > generated/spacetraders/client_gen.go
	@echo "Done!"

.PHONY: docker ## build docker image
docker:
	@echo "Building docker image..."
	@docker build -t $(PROJECT_NAME) .
	@echo "Done!"

.PHONY: lint ## run linters
lint:
	@echo "Running linters..."
	@golangci-lint run
	@echo "Done!"

# TODO: test, migrate, seed