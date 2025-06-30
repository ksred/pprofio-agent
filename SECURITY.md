# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |

## Security Considerations

### Data Security

#### Profile Data
- **Profile contents**: Profiles may contain sensitive information including:
  - Stack traces with function names and file paths
  - Memory allocations showing data structure usage
  - Goroutine stacks revealing application flow
  - Custom span data with user-defined tags

#### Recommendations
- **Review profile data**: Before enabling profiling in production, review what data is collected
- **Sanitize tags**: Avoid including sensitive information (passwords, tokens, PII) in span tags
- **Network security**: Always use HTTPS in production environments
- **API key protection**: Store API keys securely using environment variables or secret management systems

### Network Security

#### HTTPS Enforcement
The package enforces HTTPS for all API communications in production environments:
```go
// HTTPS is automatically enforced unless Env is set to "local"
storage := pprofio.NewHTTPStorage("https://api.pprofio.com/upload", apiKey, "production")
```

#### Authentication
- API keys are transmitted via Bearer token authentication
- All HTTP requests include authentication headers
- Retries preserve authentication on failure

### Access Control

#### API Keys
- API keys should be treated as sensitive credentials
- Use environment variables: `PPROFIO_API_KEY`
- Rotate keys regularly
- Monitor API key usage

#### Configuration Security
```go
// Secure configuration example
cfg := pprofio.Config{
    APIKey:      os.Getenv("PPROFIO_API_KEY"),        // From environment
    IngestURL:   "https://api.pprofio.com",           // Always HTTPS
    ServiceName: "my-service",                         // Non-sensitive identifier
    Tags: map[string]string{
        "env":     "production",                       // OK
        "version": "1.0.0",                           // OK
        // Never include: passwords, tokens, PII
    },
}
```

### Data Retention

#### Local Files
When using `FileStorage`, profiles are stored locally:
- Ensure appropriate file permissions (600/700)
- Implement log rotation for storage management
- Consider encryption for sensitive environments

#### Profile Lifecycle
- Temporary files are created during collection
- Files are automatically cleaned up after upload
- Failed uploads may leave temporary files

### Compliance Considerations

#### GDPR/Privacy
- Profiles may contain personal data in stack traces
- Consider data processing agreements with Pprofio service
- Implement data retention policies

#### Industry Standards
- Review compliance requirements for your industry
- Consider data residency requirements
- Implement audit logging if required

### Deployment Security

#### Container Security
```dockerfile
# Use non-root user
RUN adduser --disabled-password --gecos '' appuser
USER appuser

# Minimal attack surface
FROM scratch
COPY --from=builder /app/binary /binary
```

#### Kubernetes Security
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pprofio-config
type: Opaque
data:
  api-key: <base64-encoded-key>
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: app
        env:
        - name: PPROFIO_API_KEY
          valueFrom:
            secretKeyRef:
              name: pprofio-config
              key: api-key
```

## Reporting Security Vulnerabilities

If you discover a security vulnerability, please follow these steps:

1. **Do not** create a public GitHub issue
2. Send a private email to: security@pprofio.com
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Affected versions
   - Your contact information

### Response Timeline
- **Acknowledgment**: Within 24 hours
- **Initial assessment**: Within 72 hours
- **Status updates**: Weekly until resolved

## Security Best Practices

### Development
- [ ] Store API keys in environment variables
- [ ] Review profile data for sensitive information
- [ ] Use HTTPS in all non-local environments
- [ ] Sanitize custom span tags
- [ ] Implement proper error handling

### Production
- [ ] Monitor API key usage
- [ ] Implement log rotation for file storage
- [ ] Use appropriate file permissions
- [ ] Regular security audits
- [ ] Keep package updated

### Monitoring
```go
// Example: Safe span creation
ctx, span := pprofio.StartSpan(ctx, "user_action",
    "action", "login",           // OK
    "user_id", userIDHash,       // Hash sensitive IDs
    // "password", password,     // NEVER include passwords
)
defer span.End()
```

## Security Headers

The package automatically includes security-relevant headers:

- `User-Agent`: Includes version for security monitoring
- `Authorization`: Bearer token authentication
- `Content-Encoding`: Indicates compression usage

## Vulnerability Disclosure

We practice responsible disclosure:
- Security issues are addressed promptly
- Fixes are released as patch versions
- Security advisories are published when appropriate
- CVE numbers are assigned for significant vulnerabilities

For questions about security practices, contact: security@pprofio.com