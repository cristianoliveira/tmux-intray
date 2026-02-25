# CI/CD Pipeline Documentation

## Overview

tmux-intray uses GitHub Actions for continuous integration and continuous deployment. The CI/CD pipeline ensures code quality, security, and reliable releases.

## Workflows

### CI (`ci.yml`)

Runs on every push to `main` and `develop` branches, and on pull requests targeting those branches.

**Jobs:**

1. **test** - Runs Bats tests on multiple operating systems:
   - macOS latest
   - Ubuntu latest (24.04)
   - Ubuntu 22.04
   - Uses `make tests`

2. **code-quality** - Runs strict linting and security checks:
   - Lint: `make lint-strict` (includes ShellCheck, formatting check, and dependency guardrails)
   - Dependency guardrails: `make check-import-deny-rules`
   - Security: `make security-check` (security-focused ShellCheck)
   - Runs on Ubuntu latest

3. **install** - Tests installation methods on macOS:
   - npm installation
   - Go binary build
   - Source installation

4. **install-linux** - Tests installation methods on Linux:
   - npm installation
   - Go binary build
   - Source installation

### Release (`release.yml`)

Triggered when a tag matching `v[0-9]*.[0-9]*.[0-9]*` is pushed.

**Jobs:**

1. **create-release** - Creates GitHub release and artifacts:
   - Verifies version consistency between tag and binary
   - Generates documentation (man pages and CLI reference)
   - Builds Go binaries for multiple platforms (linux/amd64, linux/arm64, darwin/amd64, darwin/arm64)
   - Generates release notes from git history
   - Creates source tarball
   - Creates GitHub Release with binaries and source tarball

## How to Release

1. **Version Bump**: Ensure `bin/tmux-intray` contains the correct version in its `VERSION` variable.

2. **Create Tag**:
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

3. **Automated Process**:
   - The release workflow will:
      - Build binaries for all platforms
      - Create a GitHub Release with changelog

4. **Verify**:
   - Check the GitHub Releases page

## Dependencies

### CI Dependencies
- **macOS**: bash, bats, coreutils, shellcheck, shfmt, tmux, go, node
- **Linux**: bash, bats, coreutils, shellcheck, shfmt, tmux, golang, nodejs, npm

These are automatically installed via the `setup-environment` composite action.

### Pre-commit Hooks

Local development uses pre-commit hooks to enforce code quality before commits:
- ShellCheck on shell scripts
- shfmt formatting check
- Bats tests for changed test files

Install hooks:
```bash
pre-commit install
```

## Troubleshooting

### CI Failures

#### Test Failures
- Check Bats test output in CI logs
- Ensure tmux is available (tests require tmux server)
- Environment variables: `TMUX_INTRAY_CONFIG_LOADED=1`, `TMUX_INTRAY_DEBUG=0`

#### Lint/Security Failures
- Run `make lint-strict` locally to reproduce CI lint failures
- Run `make check-import-deny-rules` for dependency layering violations
- Run `make security-check` for security-specific warnings
- Use `make fmt` to auto-format shell scripts

#### Installation Failures
- npm installation: ensure package.json is valid
- Go build: check Go version compatibility

### Release Failures

#### Version Mismatch
Error: "Version mismatch between bin/tmux-intray and git tag"
- Ensure the `VERSION` variable in `bin/tmux-intray` matches the git tag (without leading 'v')

## Extending the Pipeline

### Adding New CI Jobs

1. Edit `.github/workflows/ci.yml`
2. Add a new job with:
   - `runs-on` specifying OS
   - `steps` using the `setup-environment` action if dependencies needed
   - Appropriate `needs` if job depends on others

### Adding New Release Artifacts

1. Edit `.github/workflows/release.yml`
2. Add build steps in `create-release` job
3. Add artifacts to the `files` list in the "Create GitHub Release" step

## Monitoring

- CI status badges in README (see below)
- GitHub Actions notifications for failures
- Dependabot alerts for security vulnerabilities

## Badges

Add these to your README.md:

```markdown
![CI](https://github.com/cristianoliveira/tmux-intray/actions/workflows/ci.yml/badge.svg)
![Release](https://github.com/cristianoliveira/tmux-intray/actions/workflows/release.yml/badge.svg)
```

## Security

- Security scanning via `make security-check`
- Dependabot configured for GitHub Actions updates
- Regular dependency updates

## Performance

- CI runs in parallel where possible
- No caching currently implemented (consider adding if build times increase)
- Average CI runtime: ~5 minutes

## Support

For CI/CD issues:
1. Check GitHub Actions logs
2. Review this documentation
3. Open an issue in the repository
