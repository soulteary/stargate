# Contributing Guide

> 🌐 **Language / 语言**: [English](CONTRIBUTING.md) | [中文](../zhCN/CONTRIBUTING.md) | [Français](../frFR/CONTRIBUTING.md) | [Italiano](../itIT/CONTRIBUTING.md) | [日本語](../jaJP/CONTRIBUTING.md) | [Deutsch](../deDE/CONTRIBUTING.md) | [한국어](../koKR/CONTRIBUTING.md)

Thank you for your interest in the Stargate project! We welcome all forms of contributions.

## 📋 Table of Contents

- [How to Contribute](#how-to-contribute)
- [Development Environment Setup](#development-environment-setup)
- [Code Standards](#code-standards)
- [Commit Standards](#commit-standards)
- [Pull Request Process](#pull-request-process)
- [Bug Reports and Feature Requests](#bug-reports-and-feature-requests)
- [Traefik Integration Development](#traefik-integration-development)

## 🚀 How to Contribute

You can contribute in the following ways:

- **Report Bugs**: Report issues in GitHub Issues
- **Suggest Features**: Propose new feature ideas in GitHub Issues
- **Submit Code**: Submit code improvements via Pull Requests
- **Improve Documentation**: Help improve project documentation
- **Answer Questions**: Help other users in Issues
- **Test Integrations**: Test Traefik integration and Warden/Herald integrations

When participating in this project, please respect all contributors, accept constructive criticism, and focus on what's best for the project.

## 🛠️ Development Environment Setup

### Prerequisites

- Go 1.25 or higher
- Redis (optional, for session storage testing)
- Git
- Traefik (optional, for integration testing)

### Quick Start

```bash
# 1. Fork and clone the project
git clone https://github.com/your-username/stargate.git
cd stargate

# 2. Add upstream repository
git remote add upstream https://github.com/soulteary/stargate.git

# 3. Install dependencies
go mod download

# 4. Run tests
go test ./...

# 5. Start local service
chmod +x start-local.sh
./start-local.sh

# Or manually
export AUTH_HOST=localhost
export PASSWORDS=plaintext:test123
go run src/cmd/stargate/main.go
```

### Testing with Traefik

For testing Traefik integration:

1. **Start Stargate**:
   ```bash
   export AUTH_HOST=auth.example.com
   export PASSWORDS=plaintext:test123
   go run src/cmd/stargate/main.go
   ```

2. **Configure Traefik** (example `traefik.yml`):
   ```yaml
   entryPoints:
     web:
       address: ":80"
   
   forwardAuth:
     address: "http://localhost:8080/_auth"
     authResponseHeaders:
       - X-Forwarded-User
   ```

3. **Test forwardAuth**: Access protected services through Traefik

### Testing with Warden and Herald

For testing service integrations:

1. **Start Warden** (if testing Warden integration)
2. **Start Herald** (if testing Herald integration)
3. **Configure Stargate**:
   ```bash
   export WARDEN_ENABLED=true
   export WARDEN_URL=http://warden:8080
   export WARDEN_API_KEY=your-api-key
   
   export HERALD_ENABLED=true
   export HERALD_URL=http://herald:8082
   export HERALD_HMAC_SECRET=your-secret
   ```

## 📝 Code Standards

Please follow these code standards:

1. **Follow Go Official Code Standards**: [Effective Go](https://go.dev/doc/effective_go)
2. **Format Code**: Run `go fmt ./...`
3. **Code Checking**: Use `golangci-lint` or `go vet ./...`
4. **Write Tests**: New features must include tests
5. **Add Comments**: Public functions and types must have documentation comments
6. **Constant Naming**: All constants must use `ALL_CAPS` (UPPER_SNAKE_CASE) naming style

### Testing Requirements

- All new features must include unit tests
- Forward auth functionality must include integration tests
- Traefik integration should be tested when possible
- Test coverage should be maintained or improved
- Run `go test ./...` before submitting PRs

## 📦 Commit Standards

### Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/) standard:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation update
- `style`: Code format adjustment (doesn't affect code execution)
- `refactor`: Code refactoring
- `perf`: Performance optimization
- `test`: Test related
- `chore`: Build process or auxiliary tool changes

### Examples

```
feat(auth): Add session sharing across domains

Implemented secure session exchange mechanism for cross-domain authentication.

Closes #123
```

```
fix(forwardauth): Fix authentication header handling

Fixed the issue where authentication headers were not properly set for API requests.

Fixes #456
```

## 🔄 Pull Request Process

### Create Pull Request

```bash
# 1. Create feature branch
git checkout -b feature/your-feature-name

# 2. Make changes and commit
git add .
git commit -m "feat: Add new feature"

# 3. Sync upstream code
git fetch upstream
git rebase upstream/main

# 4. Push branch and create PR
git push origin feature/your-feature-name
```

### Pull Request Checklist

Before submitting a Pull Request, please ensure:

- [ ] Code follows project code standards
- [ ] All tests pass (`go test ./...`)
- [ ] Code is formatted (`go fmt ./...`)
- [ ] Necessary tests are added
- [ ] Related documentation is updated
- [ ] Commit message follows [Commit Standards](#commit-standards)
- [ ] Code passes lint checks
- [ ] Traefik integration tested (if applicable)
- [ ] Warden/Herald integration tested (if applicable)

All Pull Requests require code review. Please respond to review comments promptly.

## 🐛 Bug Reports and Feature Requests

Before creating an Issue, please search existing Issues to confirm the problem or feature hasn't been reported.

### Bug Report Template

```markdown
**Description**
Clearly and concisely describe the bug.

**Reproduction Steps**
1. Execute '...'
2. See error

**Expected Behavior**
Clearly and concisely describe what you expected to happen.

**Actual Behavior**
Clearly and concisely describe what actually happened.

**Environment Information**
- OS: [e.g. macOS 12.0]
- Go Version: [e.g. 1.25]
- Traefik Version: [e.g. v2.10] (if applicable)
- Stargate Version: [e.g. v1.0.0]
```

### Feature Request Template

```markdown
**Feature Description**
Clearly and concisely describe the feature you want.

**Problem Description**
What problem does this feature solve? Why is it needed?

**Proposed Solution**
Clearly and concisely describe how you hope to implement this feature.
```

## 🔗 Traefik Integration Development

If you're working on Traefik integration features:

### Forward Auth Middleware

Stargate implements Traefik Forward Auth middleware:

- **Endpoint**: `GET /_auth`
- **Response Headers**: Sets `X-Forwarded-User` on success
- **Error Handling**: Returns 401 or redirects to login

### Testing Forward Auth

1. **Start Stargate** with test configuration
2. **Configure Traefik** to use Stargate forwardAuth
3. **Test protected routes** through Traefik
4. **Verify authentication** headers are set correctly

### Integration Examples

See `docs/enUS/DEPLOYMENT.md` for Traefik configuration examples.

## 🎯 Getting Started

If you want to contribute but don't know where to start, you can focus on:

- Issues labeled `good first issue`
- Issues labeled `help wanted`
- `TODO` comments in code
- Documentation improvements (fix typos, improve clarity, add examples)
- Test coverage improvements
- Traefik integration testing
- Warden/Herald client improvements

If you have questions, please check existing Issues and Pull Requests, or ask in relevant Issues.

---

Thank you again for contributing to the Stargate project! 🎉
