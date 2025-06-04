# Contributing to pprofio

Thank you for your interest in contributing to pprofio! This document provides guidelines and information for contributors.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [CI/CD Process](#cicd-process)
- [Code Standards](#code-standards)
- [Testing Guidelines](#testing-guidelines)
- [Release Process](#release-process)
- [Getting Help](#getting-help)

## Getting Started

### Prerequisites

- **Go**: Version 1.20 or later (we test on 1.20, 1.21, 1.22)
- **Git**: For version control
- **golangci-lint**: For code quality checks
- **Make**: For using development tasks (optional)

### Setting Up Your Development Environment

1. **Fork the repository** on GitHub
2. **Clone your fork**:
   ```bash
   git clone https://github.com/yourusername/pprofio.git
   cd pprofio
   ```
3. **Add the upstream remote**:
   ```bash
   git remote add upstream https://github.com/pprofio/pprofio.git
   ```
4. **Install dependencies**:
   ```bash
   go mod download
   ```
5. **Install development tools**:
   ```bash
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   go install golang.org/x/tools/cmd/goimports@latest
   ```

### Verifying Your Setup

Run the test suite to ensure everything is working:
```bash
go test -v -race ./...
```

Run linting to check code quality:
```bash
golangci-lint run --timeout=10m
```

## Development Workflow

### 1. Create a Feature Branch

Always create a new branch for your work:
```bash
git checkout -b feature/your-feature-name
```

Branch naming conventions:
- `feature/feature-name` - New features
- `fix/bug-description` - Bug fixes
- `docs/update-description` - Documentation updates
- `refactor/component-name` - Code refactoring

### 2. Make Your Changes

- **Follow Go conventions**: Use `gofmt`, `goimports`, and follow standard Go practices
- **Write tests**: All new code should include comprehensive tests
- **Update documentation**: Update README, godoc comments, and examples as needed
- **Keep commits focused**: Make small, logical commits with clear messages

### 3. Commit Guidelines

We follow conventional commit messages:
```
type(scope): description

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(storage): add gzip compression for HTTP uploads
fix(profiler): resolve memory leak in span collection
docs(readme): update installation instructions
test(storage): add integration tests for retry logic
```

### 4. Testing Your Changes

Before submitting, ensure all tests pass:
```bash
# Run all tests with race detection
go test -v -race ./...

# Check test coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run linting
golangci-lint run --timeout=10m

# Format code
gofmt -w .
goimports -w .

# Verify go.mod is tidy
go mod tidy
```

### 5. Submit a Pull Request

1. **Push your branch** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create a pull request** on GitHub with:
   - Clear title and description
   - Reference to any related issues
   - Screenshots/examples if applicable
   - Checklist of testing performed

## CI/CD Process

Our CI/CD pipeline automatically runs on every push and pull request:

### 🧪 Test Workflow
- **Multi-version testing**: Go 1.20, 1.21, 1.22
- **Race detection**: All tests run with `-race` flag
- **Coverage enforcement**: Minimum 80% coverage required
- **Build verification**: Ensures package compiles

### 🔍 Lint Workflow
- **Code quality**: golangci-lint with comprehensive rules
- **Security scanning**: gosec for vulnerability detection
- **Format validation**: gofmt, goimports, go mod tidy

### 📦 Release Workflow
- **Triggered by tags**: Automatic releases on version tags (v*)
- **Quality gates**: All tests and linting must pass
- **Release notes**: Auto-generated from CHANGELOG.md
- **Go module proxy**: Automatic registration and documentation updates

### Status Checks

All pull requests must pass these checks before merging:
- ✅ All tests pass on all Go versions
- ✅ Code coverage ≥ 80%
- ✅ Linting passes without errors
- ✅ Security scan passes
- ✅ Code is properly formatted

## Code Standards

### Go Style Guide

We follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) and these additional standards:

#### Naming Conventions
- **Packages**: Short, lowercase, single word when possible
- **Files**: snake_case.go
- **Functions/Methods**: CamelCase, start with uppercase for exported
- **Variables**: camelCase, descriptive names
- **Constants**: CamelCase or UPPER_CASE for package-level

#### Code Organization
```go
// Package declaration and documentation
package pprofio

// Imports grouped by: standard library, third-party, local
import (
    "context"
    "fmt"
    
    "github.com/external/package"
    
    "github.com/pprofio/pprofio/internal/util"
)

// Constants, then variables, then types, then functions
```

#### Error Handling
```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to process profile: %w", err)
}

// Use structured error types for different scenarios
type ValidationError struct {
    Field   string
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}
```

#### Documentation
```go
// Package documentation should explain purpose and usage
//
// Package pprofio provides a client library for collecting and uploading
// Go runtime profiles to the pprofio service.

// Exported functions must have documentation
//
// New creates a new Profiler instance with the given configuration.
// It validates the configuration and returns an error if invalid.
func New(config Config) (*Profiler, error) {
```

### Configuration Guidelines

- **Use struct embedding** for optional configuration
- **Provide sensible defaults** for all configuration options
- **Validate configuration** early and return clear error messages
- **Make zero values useful** when possible

### Performance Considerations

- **Minimize allocations** in hot paths
- **Use object pooling** for frequently allocated objects
- **Benchmark performance-critical code**
- **Profile memory usage** and optimize accordingly

## Testing Guidelines

### Test Organization

```
package/
├── file.go
├── file_test.go          # Unit tests
├── integration_test.go   # Integration tests
└── example_test.go       # Examples for godoc
```

### Test Categories

#### Unit Tests
- Test individual functions and methods
- Use mocks for external dependencies
- Aim for 100% coverage of business logic
- Use table-driven tests for multiple scenarios

```go
func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name    string
        config  Config
        wantErr bool
    }{
        {
            name: "valid config",
            config: Config{
                APIKey: "test-key",
                IngestURL: "https://api.pprofio.com",
            },
            wantErr: false,
        },
        // ... more test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

#### Integration Tests
- Test interactions between components
- Use real HTTP servers where appropriate
- Test with realistic data sizes
- Verify end-to-end functionality

#### Benchmarks
```go
func BenchmarkProfileCollection(b *testing.B) {
    profiler := setupProfiler()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        profiler.CollectCPUProfile(context.Background())
    }
}
```

### Test Best Practices

- **Use meaningful test names** that describe the scenario
- **Test error conditions** as well as success cases
- **Clean up resources** in tests (use defer or t.Cleanup)
- **Use testify/assert** for clear assertion messages
- **Run tests with race detection** locally before submitting

## Release Process

### Semantic Versioning

We follow [Semantic Versioning](https://semver.org/):
- **MAJOR**: Incompatible API changes
- **MINOR**: New functionality, backwards compatible
- **PATCH**: Bug fixes, backwards compatible

### Creating a Release

1. **Update CHANGELOG.md** with changes for the new version
2. **Run the release script**:
   ```bash
   ./scripts/release.sh v1.2.3
   ```
3. **Monitor GitHub Actions** for successful release creation
4. **Verify the release** appears on GitHub and pkg.go.dev

The release script will:
- Validate repository state
- Run all tests and linting
- Create and push a git tag
- Trigger GitHub Actions for release creation

### Pre-release Versions

For alpha, beta, or release candidate versions:
```bash
./scripts/release.sh v1.2.3-alpha.1
./scripts/release.sh v1.2.3-beta.1
./scripts/release.sh v1.2.3-rc.1
```

## Getting Help

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and general discussion
- **Documentation**: Check README.md and godoc
- **Code Examples**: See examples/ directory

### Issue Templates

When reporting bugs, please include:
- Go version and OS
- pprofio version
- Minimal reproduction case
- Expected vs actual behavior
- Relevant logs or stack traces

### Security Issues

For security vulnerabilities, please email security@pprofio.com instead of creating a public issue.

---

## Thank You! 🎉

Your contributions help make pprofio better for everyone. We appreciate your time and effort in improving the project! 