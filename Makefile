# Makefile for CloudFormation stack management

.PHONY: help validate deploy test clean

# Default environment and configuration
ENV ?= dev
S3_ARTIFACT_BUCKET ?=
AWS_REGION ?= us-east-1
CLOUDFORMATION_TEMPLATE ?= cloudformation.template
LAMBDA_FUNCTION_DIR ?= functions

# Default target
help: ## Show this help message
	@echo "CloudFormation Stack Management"
	@echo ""
	@echo "Usage:"
	@echo "  make <target> [ENV=<env>] [S3_ARTIFACT_BUCKET=<bucket>]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'
	@echo ""
	@echo "Environment Variables:"
	@echo "  ENV                   Environment name (default: dev)"
	@echo "  S3_ARTIFACT_BUCKET    S3 bucket for artifacts (required for deploy)"
	@echo "  AWS_REGION           AWS region (default: us-east-1)"
	@echo ""
	@echo "Examples:"
	@echo "  make validate"
	@echo "  make deploy ENV=prod S3_ARTIFACT_BUCKET=my-artifacts-bucket"
	@echo "  make test"

validate: ## Validate CloudFormation template syntax
	@echo "Validating CloudFormation template..."
	aws cloudformation validate-template \
		--template-body file://$(CLOUDFORMATION_TEMPLATE) \
		--region $(AWS_REGION)
	@echo "✓ Template is valid"

test: ## Run tests only (no deployment)
	@echo "Running tests..."
	./scripts/deploy.sh test-only

deploy: validate ## Deploy CloudFormation stack (requires ENV and S3_ARTIFACT_BUCKET)
ifndef S3_ARTIFACT_BUCKET
	$(error S3_ARTIFACT_BUCKET is required. Usage: make deploy S3_ARTIFACT_BUCKET=my-bucket ENV=dev)
endif
	@echo "Deploying CloudFormation stack..."
	@echo "Environment: $(ENV)"
	@echo "S3 Bucket: $(S3_ARTIFACT_BUCKET)"
	@echo "AWS Region: $(AWS_REGION)"
	ENV=$(ENV) S3_ARTIFACT_BUCKET=$(S3_ARTIFACT_BUCKET) AWS_REGION=$(AWS_REGION) ./scripts/deploy.sh

update: deploy ## Alias for deploy target

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	find $(LAMBDA_FUNCTION_DIR) -name "bootstrap" -delete 2>/dev/null || true
	find $(LAMBDA_FUNCTION_DIR) -name "bootstrap.zip" -delete 2>/dev/null || true
	rm -f cloudformation-params.json
	@echo "✓ Build artifacts cleaned"

lint: ## Run Go linting and formatting checks
	@echo "Running Go linting and formatting checks..."
	gofmt -s -l . | tee /dev/stderr | wc -l | grep -q "^0$$" || (echo "Code is not formatted properly. Run 'gofmt -s -w .' to fix." && exit 1)
	go vet ./...
	@echo "✓ Linting passed"

format: ## Format Go code
	@echo "Formatting Go code..."
	gofmt -s -w .
	@echo "✓ Code formatted"

build: ## Build Lambda functions locally
	@echo "Building Lambda functions..."
	@if [ ! -d "$(LAMBDA_FUNCTION_DIR)" ]; then \
		echo "Error: Lambda function directory '$(LAMBDA_FUNCTION_DIR)' not found"; \
		exit 1; \
	fi
	@for dir in $$(find $(LAMBDA_FUNCTION_DIR) -name "main.go" -type f -exec dirname {} \;); do \
		echo "Building $$dir..."; \
		cd $$dir && \
		rm -f bootstrap bootstrap.zip && \
		GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bootstrap main.go && \
		zip bootstrap.zip bootstrap && \
		cd - > /dev/null; \
	done
	@echo "✓ Lambda functions built"

status: ## Show current deployment status
	@echo "Project Status:"
	@echo "  CloudFormation Template: $(CLOUDFORMATION_TEMPLATE)"
	@echo "  Lambda Functions Dir: $(LAMBDA_FUNCTION_DIR)"
	@echo "  Environment: $(ENV)"
	@echo "  AWS Region: $(AWS_REGION)"
	@if [ -n "$(S3_ARTIFACT_BUCKET)" ]; then echo "  S3 Bucket: $(S3_ARTIFACT_BUCKET)"; fi
	@echo ""
	@echo "Lambda Functions:"
	@find $(LAMBDA_FUNCTION_DIR) -name "main.go" -type f -exec dirname {} \; | sed 's/^/  - /' || echo "  None found"