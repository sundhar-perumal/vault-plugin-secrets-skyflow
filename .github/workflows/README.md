# CI/CD Workflows

This repository uses GitHub Actions for continuous integration and delivery with a **reusable workflow architecture**.

---

## 📁 Workflow Files

### 1. **`test-coverage.yml`** (Reusable Workflow)
Centralized test and coverage logic used by both PR and release workflows.

**Features:**
- Runs tests with race detection
- Generates coverage reports (text, HTML, markdown)
- Creates PR comments with coverage (configurable)
- Publishes CTRF test reports
- Runs SonarQube analysis (configurable)
- Uploads coverage artifacts

**Parameters:**
- `go-version` - Go version to use (default: 1.23)
- `retention-days` - Artifact retention days (default: 30)
- `create-pr-comment` - Enable PR coverage comments (default: false)
- `run-sonarqube` - Enable SonarQube scan (default: false)

### 2. **`pr-check.yml`** (PR Validation)
Runs on pull requests to validate code quality.

**Jobs:**
- **Lint** - Code quality checks with golangci-lint
- **Test** - Calls reusable test-coverage workflow (30-day retention, PR comments ON, SonarQube ON)
- **Security** - Security scanning with Gosec + SARIF
- **Build** - Verify compilation

### 3. **`release.yml`** (Release Automation)
Runs on version tags (`v*.*.*`) to create releases.

**Jobs:**
- **Validate** - Pre-release checks (lint, security, build)
- **Test** - Calls reusable test-coverage workflow (90-day retention, PR comments OFF, SonarQube ON)
- **Release** - Create GitHub release with auto-generated changelog + coverage HTML
- **Publish** - Verify Go module publication

---

## 🎯 Key Features

### ✅ **Reusable Architecture**
- Test/coverage/SonarQube logic defined once in `test-coverage.yml`
- Both PR and release workflows call the same reusable workflow
- DRY principle - changes in one place update everywhere

### ✅ **Private Repository Support**
- Git URL rewriting for `github.com/angel-one/*` repos
- Uses `SRE_GIT_READ_TOKEN` for authentication
- Explicit `shell: bash` for cross-platform compatibility

### ✅ **Coverage Strategy (No Codecov License Required)**
- **PR Comments** - Automatic coverage reports in pull requests
- **Artifacts** - Downloadable HTML and raw coverage files
- **Console Output** - Coverage percentage in workflow logs
- **Release Assets** - Coverage HTML attached to GitHub releases

### ✅ **Quality Gates**
- Linting with golangci-lint
- Race detection on all tests
- Security scanning with Gosec + GitHub Security tab
- SonarQube code quality analysis (optional)
- Build validation

### ✅ **Test Reporting**
- **CTRF Format** - Visual test results in PRs
- **Coverage Reports** - Detailed package-level coverage
- **Downloadable Artifacts** - HTML reports for offline viewing

### ✅ **Release Automation**
- **GitHub Auto-Release Notes** - Generated from PR history
- **Semantic Versioning** - Manual tag creation (`v1.2.3`)
- **Automatic Changelog** - Grouped by PR labels
- **Artifact Attachment** - Coverage HTML included in releases

---

## 🔧 Configuration

### Required Secrets
- `SRE_GIT_READ_TOKEN` - Token for accessing private Angel One repositories
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

### Optional Secrets (SonarQube)
- `SONAR_TOKEN` - SonarQube authentication token
- `SONAR_HOST_URL` - Your SonarQube server URL

### Optional (Future - Codecov)
- `CODECOV_TOKEN` - Uncomment sections in `test-coverage.yml` when available

---

## 🚀 Usage

### For Pull Requests
1. Create a feature branch
2. Make changes and commit
3. Open PR to `main`, `master`, or `hotfix-*`
4. Workflow automatically runs:
   - Linting
   - Tests with race detection
   - Coverage with PR comments
   - CTRF test reports
   - Security scanning
   - SonarQube analysis
   - Build verification

### For Releases
1. **Decide version number** based on changes:
   - `v1.0.1` - Bug fixes (PATCH)
   - `v1.1.0` - New features (MINOR)
   - `v2.0.0` - Breaking changes (MAJOR)

2. **Create and push tag:**
   ```bash
   git tag v1.2.3
   git push origin v1.2.3
   ```

3. **Workflow automatically:**
   - Validates code (lint, security, build)
   - Runs full test suite
   - Generates coverage reports
   - Creates GitHub release with auto-generated notes
   - Attaches coverage HTML
   - Verifies Go module publication

---

## 📊 Workflow Comparison

| Feature | PR Check | Release |
|---------|----------|---------|
| **Trigger** | Pull requests | Version tags (v*.*.*) |
| **Coverage Retention** | 30 days | 90 days |
| **PR Comments** | ✅ Yes | ❌ No |
| **SonarQube** | ✅ Yes (if configured) | ✅ Yes (if configured) |
| **Separate Jobs** | Lint, Test, Security, Build | Validate, Test, Release, Publish |
| **Artifacts** | Coverage reports | Coverage + Release |

---

## 🔍 SonarQube Integration

### Setup (Optional)
1. Create project in SonarQube
2. Update `sonar-project.properties` with your project key
3. Add secrets to GitHub:
   - `SONAR_TOKEN`
   - `SONAR_HOST_URL`

### How It Works
- Automatically runs after tests (needs coverage data)
- Only runs if `SONAR_TOKEN` is set
- Conditionally enabled via `run-sonarqube` parameter
- Same logic for both PR and release workflows

### Disable SonarQube
Remove secrets or set `run-sonarqube: false` in workflow calls.

---

## 📝 GitHub Auto-Generated Release Notes

The release workflow uses GitHub's built-in release notes generator:

```yaml
generate_release_notes: true
```

### How It Works
- Automatically creates changelog from merged PRs
- Groups by PR labels (feature, bug, documentation, etc.)
- Credits all contributors
- Links to PRs for context
- Highlights first-time contributors

### Make It Better - Use PR Labels
| Label | Section in Release Notes |
|-------|-------------------------|
| `enhancement`, `feature` | ✨ New Features |
| `bug`, `bugfix` | 🐛 Bug Fixes |
| `documentation`, `docs` | 📚 Documentation |
| `security` | 🔒 Security |
| No label | Other Changes |

**Best Practice:** Label your PRs before merging!

---

## 🛠️ Local Development

### Run Tests Locally
```bash
# Basic tests
go test -v ./...

# With race detection and coverage (matches CI)
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...

# View coverage
go tool cover -func=coverage.out
go tool cover -html=coverage.out
```

### Run Linting Locally
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linting
golangci-lint run --timeout=5m
```

### Run Security Scan Locally
```bash
# Install gosec
go install github.com/securego/gosec/v2/cmd/gosec@latest

# Run security scan
gosec ./...
```

---

## 🎓 Semantic Versioning Guide

### Version Format: `vMAJOR.MINOR.PATCH`

**PATCH (v1.0.X)** - Bug fixes only
- ✅ Fixed a bug
- ✅ Security patch
- ✅ Performance improvement (no API changes)
- ✅ Documentation fixes
- **Decision:** Users can upgrade without code changes

**MINOR (v1.X.0)** - New features (backward compatible)
- ✅ Added new function/method
- ✅ Added optional parameter
- ✅ New configuration option
- ✅ Deprecated old function (still works)
- **Decision:** Users can upgrade and use new features, old code still works

**MAJOR (vX.0.0)** - Breaking changes
- 🚨 Removed function/method
- 🚨 Changed function signature
- 🚨 Renamed exported types
- 🚨 Changed required configuration
- **Decision:** Users MUST update their code to upgrade

---

## 📦 Files Overview

```
.github/workflows/
├── test-coverage.yml    # Reusable: Test + Coverage + SonarQube
├── pr-check.yml         # PR validation workflow
├── release.yml          # Release automation workflow
└── README.md            # This file
```

---

## 🔗 Related Files

- `sonar-project.properties` - SonarQube configuration
- `.gitignore` - Git ignore patterns
- `go.mod` - Go module dependencies

---

## 💡 Future Enhancements

### Enable Codecov (When License Available)
Uncomment these lines in `test-coverage.yml`:
```yaml
- name: Upload to Codecov
  uses: codecov/codecov-action@v4
  with:
    file: ./coverage.out
    flags: unittests
    name: codecov-umbrella
```

### Customize Release Notes
Create `.github/release.yml` for advanced grouping:
```yaml
changelog:
  categories:
    - title: 🚀 Features
      labels:
        - feature
        - enhancement
    - title: 🐛 Bug Fixes
      labels:
        - bug
```

---

## 🎯 Benefits of This Architecture

| Benefit | Description |
|---------|-------------|
| **DRY** | Test/coverage logic defined once, used everywhere |
| **Maintainable** | Changes in one place update all workflows |
| **Flexible** | Parameters allow customization per use case |
| **Scalable** | Easy to add more workflows that need testing |
| **Clean** | Main workflows are simple and readable |
| **No External Deps** | Coverage handled in-house (no Codecov needed yet) |

---

## 📞 Support

For issues or questions:
1. Check workflow run logs in GitHub Actions tab
2. Review this documentation
3. Contact DevOps/SRE team

---

**Last Updated:** December 2025

