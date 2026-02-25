# Contributing to RxFlow

Thank you for your interest in contributing to RxFlow! This document provides guidelines and information for contributors.

## Code of Conduct

By participating in this project, you agree to maintain a respectful and inclusive environment for everyone.

## Getting Started

### Prerequisites

- Go 1.23 or later
- Docker and Docker Compose
- Make
- Git

### Setting Up Development Environment

1. **Fork and clone the repository**
   ```bash
   git clone https://github.com/yourusername/go-oec.git
   cd go-oec
   ```

2. **Start the infrastructure**
   ```bash
   make docker-up
   ```

3. **Run database migrations**
   ```bash
   make migrate-up
   ```

4. **Build and test**
   ```bash
   make build
   make test
   ```

## Development Workflow

### Branching Strategy

We follow Git Flow:

- `main` - Production-ready code
- `develop` - Integration branch for features
- `feature/*` - New features
- `fix/*` - Bug fixes
- `release/*` - Release preparation

### Making Changes

1. Create a feature branch from `develop`:
   ```bash
   git checkout develop
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the coding standards below

3. Write or update tests as needed

4. Run the test suite:
   ```bash
   make test
   make lint
   ```

5. Commit your changes with meaningful messages:
   ```bash
   git commit -m "Add feature: description of the change"
   ```

6. Push to your fork and create a Pull Request

## Coding Standards

### Go Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go) guidelines
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Keep functions focused and under 50 lines when possible

### Naming Conventions

- Use `camelCase` for local variables and unexported functions
- Use `PascalCase` for exported functions, types, and constants
- Use descriptive names that reflect purpose

### Error Handling

- Always handle errors explicitly
- Use custom error types for domain errors
- Include context in error messages
- Don't use `panic` in library code

### Documentation

- Document all exported functions, types, and constants
- Use complete sentences in documentation
- Include examples for complex functionality

### Testing

- Write table-driven tests when appropriate
- Aim for 80%+ code coverage
- Include both unit and integration tests
- Test edge cases and error conditions

## Pull Request Guidelines

### Before Submitting

- [ ] Code compiles without errors
- [ ] All tests pass (`make test`)
- [ ] Linter passes (`make lint`)
- [ ] Documentation is updated
- [ ] Commit messages are clear and descriptive

### PR Description

Include:
- Summary of changes
- Related issue numbers
- Testing performed
- Breaking changes (if any)

### Review Process

1. PRs require at least one approving review
2. Address all review comments
3. Keep PRs focused and reasonably sized
4. Squash commits before merging if needed

## Issue Reporting

### Bug Reports

Include:
- Clear description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)
- Relevant logs or error messages

### Feature Requests

Include:
- Use case description
- Proposed solution (if any)
- Alternatives considered
- Impact on existing functionality

## Architecture Guidelines

### Domain-Driven Design

- Keep domain logic in `internal/domain/`
- Use aggregates for transactional consistency
- Events should be immutable
- Commands should validate input

### FHIR/NCPDP Standards

- Follow FHIR R5 specification strictly
- Follow NCPDP SCRIPT v2023011 specification
- Document any standard deviations

### Security

- Never log sensitive data (PHI, credentials)
- Use prepared statements for SQL
- Validate all external input
- Follow OWASP guidelines

## Questions?

- Open an issue for general questions
- Tag maintainers for urgent matters
- Check existing issues before creating new ones

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
