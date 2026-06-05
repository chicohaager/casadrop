# Contributing to CasaDrop

Thank you for your interest in contributing to CasaDrop!

## Development Setup

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- Git

### Local Development

```bash
# Clone the repository
git clone https://github.com/user/casadrop.git
cd casadrop

# Run locally (requires Go)
go run ./cmd/server

# Or use Docker
docker compose up -d
```

### Project Structure

```
casadrop/
├── cmd/server/          # Main entry point
├── internal/
│   ├── auth/            # OIDC authentication
│   ├── handlers/        # HTTP handlers
│   ├── middleware/      # Auth, rate-limit, security
│   ├── models/          # Data models
│   ├── storage/         # SQLite storage
│   └── webhook/         # Webhook notifications
├── web/
│   ├── static/          # CSS, JS, images
│   └── templates/       # HTML templates
├── docs/                # Documentation
└── .github/workflows/   # CI/CD pipelines
```

## Making Changes

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

### 2. Make Your Changes

- Follow existing code style
- Add comments for complex logic
- Update documentation if needed

### 3. Test Your Changes

```bash
# Build and run
docker build -t casadrop:test .
docker run -p 8080:8080 casadrop:test

# Check the app works
curl http://localhost:8080/api/auth/status
```

### 4. Submit a Pull Request

- Write a clear PR description
- Reference any related issues
- Ensure CI checks pass

## Code Style

- Go: Follow standard Go conventions (`gofmt`, `go vet`)
- JavaScript: Vanilla JS, no frameworks
- CSS: Use CSS variables for theming
- HTML: Semantic HTML5

## Commit Messages

Use conventional commits:

```
feat: add OIDC authentication support
fix: resolve nil pointer in status handler
docs: update README with Docker instructions
chore: update dependencies
```

## Reporting Issues

When reporting bugs, please include:

1. CasaDrop version
2. Docker/OS version
3. Steps to reproduce
4. Expected vs actual behavior
5. Relevant logs

## Feature Requests

Open an issue with:

1. Clear description of the feature
2. Use case / why it's needed
3. Any implementation ideas

## Security Issues

For security vulnerabilities, please email directly instead of opening a public issue.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
