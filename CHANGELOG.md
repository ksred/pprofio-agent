# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GitHub Actions CI/CD workflows for automated testing and releases
- Comprehensive golangci-lint configuration for code quality
- Multi-version Go testing (1.20, 1.21, 1.22)
- Race detection in all tests
- Code coverage enforcement (80% minimum)
- Security scanning with gosec
- Automated release management with semantic versioning

### Changed
- Updated project structure for Go package distribution best practices
- Improved code formatting and import organization

### Deprecated
- Nothing currently deprecated

### Removed
- Nothing currently removed

### Fixed
- Linting configuration updated for latest golangci-lint version
- Removed deprecated linters from configuration

### Security
- Added security scanning to CI pipeline
- Enforced HTTPS in workflows where applicable

---

## Template for Future Releases

### [X.Y.Z] - YYYY-MM-DD

### Added
- New features added to the profiler
- New configuration options
- New profile types supported

### Changed
- Changes in existing functionality
- API modifications (breaking changes should bump major version)
- Performance improvements

### Deprecated
- Features that will be removed in future versions
- API methods that are being phased out

### Removed
- Features removed in this release
- APIs that were previously deprecated

### Fixed
- Bug fixes
- Performance improvements
- Security vulnerabilities patched

### Security
- Security-related changes
- Vulnerability fixes

---

## Release Guidelines

When creating a new release:

1. **Update this CHANGELOG.md** with all changes since the last release
2. **Follow semantic versioning**:
   - **MAJOR** version when you make incompatible API changes
   - **MINOR** version when you add functionality in a backwards compatible manner
   - **PATCH** version when you make backwards compatible bug fixes

3. **Tag the release** with format `vX.Y.Z` (e.g., `v1.2.3`)
4. **Let GitHub Actions handle the release creation** automatically

### Version History

This section will be populated as releases are made.

<!-- 
Example format for releases:
## [1.0.0] - 2024-01-15
### Added
- Initial stable release of pprofio Go profiling client
- CPU profile collection with configurable sample rates
- Memory profile collection (heap, allocs)
- Goroutine profile collection
- Mutex contention profiling
- Block profiling support
- Custom span instrumentation API
- HTTP and File storage backends
- Comprehensive test suite with >90% coverage

### Security
- HTTPS enforcement for all HTTP communications
- API key authentication for SaaS uploads
--> 