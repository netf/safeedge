---
description: Prepare a pull request with linting, formatting, tests, and commit staging
---

Prepare the current branch for a pull request by running all quality checks and staging changes.

Steps:
1. Run `go fmt ./...` to format all Go code
2. Run `go vet ./...` to check for common mistakes
3. If `internal/controlplane/database/queries/` changed, run `cd internal/controlplane/database && sqlc generate`
4. Run `go test ./...` to ensure tests pass (if tests exist)
5. Run `git status` to show current changes
6. Ask the user for a commit message
7. Stage changes with `git add .`
8. Create commit with the provided message
9. Provide next steps for pushing and creating the PR
