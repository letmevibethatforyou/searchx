#!/bin/bash

set -euo pipefail

# Configuration
GO_VERSION="1.24"
LAMBDA_FUNCTION_DIR="${LAMBDA_FUNCTION_DIR:-functions}"
CLOUDFORMATION_TEMPLATE="${CLOUDFORMATION_TEMPLATE:-cloudformation.template}"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

usage() {
    cat << EOF
Usage: $0 [test-only]

Deploy searchx Lambda functions to AWS using the AWS Deployer pattern.

ARGUMENTS:
    test-only    Run tests only, skip deployment

ENVIRONMENT VARIABLES:
    S3_ARTIFACT_BUCKET (required)  S3 bucket for artifacts
    ENV (required)                 Environment name
    VERSION                        Version string (default: auto-generated)
    AWS_REGION                     AWS region (default: us-east-1)
    LAMBDA_FUNCTION_DIR            Lambda function directory (default: functions)
    CLOUDFORMATION_TEMPLATE        CloudFormation template file (default: cloudformation.template)

EXAMPLES:
    S3_ARTIFACT_BUCKET=my-bucket ENV=prod ./deploy.sh
    S3_ARTIFACT_BUCKET=my-bucket ENV=dev ./deploy.sh
    ./deploy.sh test-only
EOF
}

check_requirements() {
    log_info "Checking requirements..."

    # Check if go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi

    # Check if aws cli is installed
    if ! command -v aws &> /dev/null; then
        log_error "AWS CLI is not installed or not in PATH"
        exit 1
    fi

    # Check if zip is installed
    if ! command -v zip &> /dev/null; then
        log_error "zip is not installed or not in PATH"
        exit 1
    fi

    # Check Go version
    GO_CURRENT_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
    log_info "Go version: $GO_CURRENT_VERSION"

    # Check if Lambda function directory exists
    if [ ! -d "$LAMBDA_FUNCTION_DIR" ]; then
        log_error "Lambda function directory not found: $LAMBDA_FUNCTION_DIR"
        exit 1
    fi

    # Check if CloudFormation template exists
    if [ ! -f "$CLOUDFORMATION_TEMPLATE" ]; then
        log_error "CloudFormation template not found: $CLOUDFORMATION_TEMPLATE"
        exit 1
    fi

    log_info "All requirements satisfied"
}

run_tests() {
    log_info "Running tests..."

    # Download dependencies
    log_info "Downloading Go modules..."
    go mod download

    # Run tests
    log_info "Running go test..."
    go test -v ./...

    # Run go vet
    log_info "Running go vet..."
    go vet ./...

    # Check formatting
    log_info "Checking code formatting..."
    UNFORMATTED=$(gofmt -s -l . | wc -l)
    if [ "$UNFORMATTED" -gt 0 ]; then
        log_error "The following files are not formatted properly:"
        gofmt -s -l .
        log_error "Run 'gofmt -s -w .' to fix formatting issues"
        exit 1
    fi

    log_info "All tests passed"
}

generate_version() {
    if [ -n "${VERSION:-}" ]; then
        echo "$VERSION"
        return
    fi

    # Use GitHub run number if available, otherwise fallback to timestamp
    if [ -n "${GITHUB_RUN_NUMBER:-}" ]; then
        VERSION_PREFIX="${GITHUB_RUN_NUMBER}"
    else
        VERSION_PREFIX=$(date +%Y%m%d%H%M%S)
    fi

    if command -v git &> /dev/null && [ -d .git ]; then
        GIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "nogit")
    else
        GIT_HASH="nogit"
    fi

    echo "${VERSION_PREFIX}.${GIT_HASH}"
}

build_lambda() {
    log_info "Building Lambda function..."

    # Find Lambda function directories
    LAMBDA_DIRS=$(find "$LAMBDA_FUNCTION_DIR" -name "main.go" -type f -exec dirname {} \;)

    if [ -z "$LAMBDA_DIRS" ]; then
        log_error "No Lambda functions found in $LAMBDA_FUNCTION_DIR"
        exit 1
    fi

    for lambda_dir in $LAMBDA_DIRS; do
        log_info "Building Lambda function in $lambda_dir"

        cd "$lambda_dir"

        # Clean previous builds
        rm -f bootstrap bootstrap.zip

        # Build for Lambda (Linux AMD64)
        log_info "Compiling Go binary for Lambda runtime..."
        GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bootstrap main.go

        if [ ! -f bootstrap ]; then
            log_error "Failed to build Lambda function in $lambda_dir"
            exit 1
        fi

        # Create zip file
        log_info "Creating bootstrap.zip..."
        zip bootstrap.zip bootstrap

        if [ ! -f bootstrap.zip ]; then
            log_error "Failed to create bootstrap.zip in $lambda_dir"
            exit 1
        fi

        log_info "Lambda function built successfully in $lambda_dir"

        # Return to root directory
        cd - > /dev/null
    done
}

upload_artifacts() {
    local version="$1"
    local env="$2"
    local bucket="$3"

    log_info "Uploading artifacts for version $version..."

    # Get repository name from current directory or git
    if command -v git &> /dev/null && [ -d .git ]; then
        REPO_NAME=$(basename "$(git rev-parse --show-toplevel)")
        BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
    else
        REPO_NAME=$(basename "$(pwd)")
        BRANCH_NAME="unknown"
    fi

    S3_PREFIX="${REPO_NAME}/${BRANCH_NAME}/${version}"

    log_info "S3 prefix: $S3_PREFIX"

    # Upload Lambda functions
    LAMBDA_DIRS=$(find "$LAMBDA_FUNCTION_DIR" -name "bootstrap.zip" -type f -exec dirname {} \;)

    for lambda_dir in $LAMBDA_DIRS; do
        LAMBDA_NAME=$(basename "$lambda_dir")
        log_info "Uploading $LAMBDA_NAME..."
        log_info "aws s3 cp ${lambda_dir}/bootstrap.zip s3://${S3_ARTIFACT_BUCKET}/${S3_PREFIX}/${LAMBDA_NAME}.zip"
        aws s3 cp "${lambda_dir}/bootstrap.zip" "s3://${S3_ARTIFACT_BUCKET}/${S3_PREFIX}/${LAMBDA_NAME}.zip"
    done

    # Upload CloudFormation template
    log_info "Uploading CloudFormation template..."
    aws s3 cp "$CLOUDFORMATION_TEMPLATE" "s3://${S3_ARTIFACT_BUCKET}/${S3_PREFIX}/cloudformation.template"

    # Create and upload CloudFormation parameters
    echo -e "${YELLOW}Creating CloudFormation parameters file...${NC}"
    cat > cloudformation-params.json << EOF
[
  {
    "ParameterKey": "S3BucketName",
    "ParameterValue": "${S3_ARTIFACT_BUCKET}"
  },
  {
    "ParameterKey": "S3KeyPrefix",
    "ParameterValue": "${S3_PREFIX}"
  },
  {
    "ParameterKey": "Version",
    "ParameterValue": "${version}"
  },
  {
    "ParameterKey": "Env",
    "ParameterValue": "${env}"
  },
  {
    "ParameterKey": "AlgoliaSecretArn",
    "ParameterValue": "${ALGOLIA_SECRET_ARN:=}"
  },
  {
    "ParameterKey": "Repository",
    "ParameterValue": "${REPO_NAME:=}"
  }
]
EOF

    log_info "Generated parameters:"
    cat cloudformation-params.json

    # Upload parameters file (this triggers deployment)
    log_info "Uploading parameters file (this will trigger deployment)..."
    aws s3 cp cloudformation-params.json "s3://${S3_ARTIFACT_BUCKET}/${S3_PREFIX}/cloudformation-params.json"

    # Clean up local parameters file
    rm cloudformation-params.json

    log_info "Deployment triggered successfully!"
    log_info "Version: $version"
    log_info "Environment: $env"
    log_info "Artifacts location: s3://${S3_ARTIFACT_BUCKET}/${S3_PREFIX}"
}

main() {
    # Check for help
    if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
        usage
        exit 0
    fi

    # Check for test-only mode
    TEST_ONLY=false
    if [[ "${1:-}" == "test-only" ]]; then
        TEST_ONLY=true
    fi

    # Get values from environment variables
    BUCKET="${S3_ARTIFACT_BUCKET:-}"
    ENV="${ENV:-}"
    REGION="${AWS_REGION:-us-east-1}"

    # Set AWS region
    export AWS_DEFAULT_REGION="$REGION"

    # Validate required parameters
    if [ -z "$BUCKET" ] && [ "$TEST_ONLY" != "true" ]; then
        log_error "S3_ARTIFACT_BUCKET environment variable is required for deployment."
        usage
        exit 1
    fi

    if [ -z "$ENV" ] && [ "$TEST_ONLY" != "true" ]; then
        log_error "ENV environment variable is required for deployment."
        usage
        exit 1
    fi

    log_info "Starting deployment process..."
    if [ "$TEST_ONLY" != "true" ]; then
        log_info "Environment: $ENV"
        log_info "S3 Bucket: $BUCKET"
    fi
    log_info "AWS Region: $REGION"
    log_info "Lambda functions directory: $LAMBDA_FUNCTION_DIR"
    log_info "CloudFormation template: $CLOUDFORMATION_TEMPLATE"

    # Check requirements
    check_requirements

    # Run tests
    run_tests

    # Exit if test-only mode
    if [ "$TEST_ONLY" = "true" ]; then
        log_info "Test-only mode: skipping deployment"
        exit 0
    fi

    # Generate version
    VERSION=$(generate_version)
    log_info "Deployment version: $VERSION"

    # Build Lambda function
    build_lambda

    # Upload artifacts and trigger deployment
    upload_artifacts "$VERSION" "$ENV" "$BUCKET"

    log_info "Deployment completed successfully!"
}

# Run main function with all arguments
main "$@"