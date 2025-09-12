# Build the provider
.PHONY: build
build:
	go build -o terraform-provider-auth0-connections

# Install the provider locally
.PHONY: install
install: build
	mkdir -p ~/.terraform.d/plugins/registry.terraform.io/cerifi/auth0-connections/1.0.0/linux_amd64
	cp terraform-provider-auth0-connections ~/.terraform.d/plugins/registry.terraform.io/cerifi/auth0-connections/1.0.0/linux_amd64/

# Run tests
.PHONY: test
test:
	go test ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test -v -cover ./...

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	golangci-lint run

# Clean build artifacts
.PHONY: clean
clean:
	rm -f terraform-provider-auth0-connections

# Generate documentation
.PHONY: docs
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m
