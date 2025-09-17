# Repository Guidelines

## Project Structure & Module Organization
Core abstractions live at the repository root (`searcher.go`, `types.go`, `options.go`). Backends are isolated in `algolia/` and `inmemory/`, with DynamoDB stream helpers under `internal/ddb/`. Command-line utilities and data generators reside in `cmd/`, while deployable Lambda handlers sit in `functions/` (one `main.go` per function). Cloud resources are described in `cloudformation.template`, and supporting automation is grouped under `scripts/`.

## Build, Test, and Development Commands
- `go test ./...` runs the full Go unit suite locally.
- `make test` triggers the deploy script in test mode (`go mod download`, `go test`, `go vet`, and `gofmt` checks).
- `make lint` enforces formatting and vetting without touching binaries.
- `make build` cross-compiles every Lambda in `functions/` and zips the outputs.
- `./scripts/deploy.sh` (with `ENV` and `S3_ARTIFACT_BUCKET`) validates, builds, and uploads CloudFormation artifacts; call it via the Makefile when possible.

## Coding Style & Naming Conventions
Format Go code with `gofmt -s`; tabs are standard and imports should stay grouped. Packages use short lowercase names, while exported identifiers need GoDoc-style comments. Prefer descriptive variable names, keep error values in the `ErrXYZ` pattern, and wrap context with `fmt.Errorf`.

## Testing Guidelines
Place tests beside implementations as `*_test.go` files and match package names. Follow table-driven patterns where practical, especially for search options and DynamoDB decoding. Run `go test ./...` before submitting and ensure `make test` succeeds so vetting and formatting gates stay green. Mock external services with the in-memory searcher or lightweight stubs rather than calling live APIs.

## Commit & Pull Request Guidelines
Commits follow a single-sentence, imperative subject line (e.g., `Configure JSON logging for AWS environments`). Keep subjects under roughly 65 characters and add body paragraphs when extra context or rollback notes matter. Pull requests should describe behavior changes, note `go test ./...` results, and link relevant issues. Include screenshots or logs when altering deployment flows or CLI output.

## AWS Deployment & Configuration Tips
Rely on the Makefile for CloudFormation tasks (`make validate`, `make deploy`). Always supply `ENV`, `S3_ARTIFACT_BUCKET`, and optionally `AWS_REGION`. Store Algolia credentials in AWS Secrets Manager and let `algolia/aws.go` resolve them; never commit raw keys. Clean Lambda build artifacts with `make clean` before packaging another release cycle.
