# Contributing to ⛵ Gorrent

First off, thank you for considering contributing to Gorrent! It's people like you that make open source such a great community.

Gorrent is an open source project licensed under **GPL v3**. Even though it is currently maintained primarily by a single developer, contributions of any kind (bug reports, feature requests, documentation improvements, and pull requests) are warmly welcomed.

This document outlines the development setup, conventions, and tips to help you contribute efficiently.

## Contact

If you have any questions, want to discuss a new feature, or need help understanding the codebase, feel free to reach out via [GitHub Issues](https://github.com/x-name15/gorrent/issues) or Discussions.

## Development Workflow

Here is the suggested workflow for contributing code:

1. **Fork and Clone:** Fork the repository on GitHub and clone it locally to your machine.
2. **Environment Setup:** Set up your Go environment (see requirements below).
3. **Branching:** Create a topic branch from `main` for your contribution (e.g., `git checkout -b feature/postgres-export`).
4. **Implementation:** Write your code.
   - Keep your commits logical and atomic.
   - Ensure you follow the commit message conventions below.
   - Add new tests if you are adding new features.
5. **Testing:** Run the test suite and ensure all tests pass (including the new ones you wrote).
6. **Push:** Push your topic branch to your GitHub fork.
7. **Pull Request:** Submit a Pull Request (PR) to the `main` branch of the original repository.
8. **Review:** Address any review comments. Once approved, your PR will be merged!

## Setting up the Development Environment

### Requirements

- Go 1.25+
- Docker & Docker Compose (for integration testing and cross-compilation)

A [`.devcontainer`](./.devcontainer) is included if you'd rather not set any of this up locally — open the repo in VS Code and "Reopen in Container."

### Build & Run Locally

To build the executables and run the server locally without Docker:

```bash
# Get dependencies
go mod tidy

# Build the server and CLI
go build -o bin/gorrent-daemon ./cmd/daemon
go build -o bin/gorrent ./cmd/cli

# Run the server (starts on port 7800 by default)
./bin/gorrent-daemon
```

### Testing

Testing is a critical part of Gorrent's reliability.

- **Unit tests:** Verify individual functions and scrapers.
- **Integration tests:** Verify engine capabilities and P2P integrations.

To run the full test suite locally:

```bash
# Run all tests with the race detector
go test -race -timeout 30s ./internal/tests/...

# Run performance benchmarks
go test -bench=. -benchmem ./internal/benchmarks/...
```

To run end-to-end (E2E) integration tests using Docker (recommended before opening a PR):

```bash
docker compose up --build -d
```

## Commit Conventions

We prefer clear, descriptive commit messages. A good commit message helps reviewers understand why a change was made.

**Format:**

```text
Verb feature-name: Short description of what changed

Detailed explanation of why this change was necessary.
Explain the problem it solves or the architectural decision made.
```

**Example:**

```text
Add bound checks to storage field decoding

Length-prefixed fields in log.go/snapshot.go were allocated before
being validated, so a corrupted or truncated file could force an
unbounded make([]byte, n)) ahead of the CRC32 check. Added hard
ceilings per field, checked before allocation.
```

- Subject line should be ≤ 70 characters.
- Use the imperative mood ("Add feature" not "Added feature").
- Leave a blank line after the subject.

## Code Review

All contributions are reviewed before merging. Even though the project is small, pull requests should include:

- A clear description of the change and the problem it solves.
- Relevant unit or integration tests.
- Code formatted with `gofmt` (`go fmt ./...`).

Happy coding, and thank you for helping make Gorrent more solid! ⛵