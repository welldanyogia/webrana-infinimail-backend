# Release Process Guide

Ship with confidence. This guide describes the complete release process for Infinimail backend.

## Table of Contents

- [Release Types](#release-types)
- [Semantic Versioning](#semantic-versioning)
- [Pre-Release Checklist](#pre-release-checklist)
- [Creating a Release](#creating-a-release)
- [Post-Release Verification](#post-release-verification)
- [Rollback Procedure](#rollback-procedure)
- [Troubleshooting](#troubleshooting)

## Release Types

### Stable Releases

Production-ready releases following semantic versioning.

**Format:** `vX.Y.Z` (e.g., `v1.2.3`)

**When to use:**
- New features ready for production
- Bug fixes for production issues
- Security patches

### Pre-Releases

Test releases for validation before stable release.

**Formats:**
- **Alpha:** `vX.Y.Z-alpha.N` (e.g., `v1.3.0-alpha.1`)
- **Beta:** `vX.Y.Z-beta.N` (e.g., `v1.3.0-beta.1`)
- **Release Candidate:** `vX.Y.Z-rc.N` (e.g., `v1.3.0-rc.1`)

**When to use:**
- Alpha: Internal testing, unstable features
- Beta: External testing, feature complete
- RC: Final validation before stable

## Semantic Versioning

We follow [Semantic Versioning 2.0.0](https://semver.org/).

### Version Format: MAJOR.MINOR.PATCH

```
v1.2.3
│ │ │
│ │ └─ PATCH: Bug fixes, backward compatible
│ └─── MINOR: New features, backward compatible
└───── MAJOR: Breaking changes, incompatible API changes
```

### When to Increment

**MAJOR (Breaking Changes):**
- Incompatible API changes
- Removed or renamed endpoints
- Changed request/response formats
- Database schema changes requiring migration
- Configuration format changes

Examples: `v1.0.0` → `v2.0.0`

**MINOR (New Features):**
- New API endpoints
- New optional parameters
- New optional features
- Backward-compatible enhancements
- Performance improvements

Examples: `v1.2.0` → `v1.3.0`

**PATCH (Bug Fixes):**
- Bug fixes
- Security patches
- Documentation updates
- Internal refactoring
- Dependency updates (security)

Examples: `v1.2.3` → `v1.2.4`

## Pre-Release Checklist

Complete ALL items before creating a release tag.

### Code Quality

- [ ] All tests passing locally (`make test`)
- [ ] No linting errors (`make lint`)
- [ ] Code formatted (`make fmt`)
- [ ] Test coverage >= 70% (`make test-coverage`)
- [ ] Race detector passed (`make test-race`)

### Documentation

- [ ] CHANGELOG.md updated with changes
- [ ] README.md updated if needed
- [ ] API documentation updated (if API changed)
- [ ] Migration guide created (if breaking changes)
- [ ] Version bumped in relevant files

### Security

- [ ] No secrets in code or config files
- [ ] Security vulnerabilities addressed
- [ ] Dependencies updated (critical security patches)
- [ ] .env.secure.example updated with new variables

### Database

- [ ] Migration scripts tested
- [ ] Rollback scripts prepared
- [ ] Backup procedure verified
- [ ] Schema changes documented

### Integration

- [ ] Integration tests passing
- [ ] E2E tests passing
- [ ] API contract tests passing
- [ ] Tested with frontend (if applicable)

### Infrastructure

- [ ] Docker image builds successfully
- [ ] docker-compose.prod.yml tested
- [ ] Environment variables documented
- [ ] Resource requirements updated

## Creating a Release

### Step 1: Prepare the Release Branch

```bash
# Ensure you're on main and up to date
git checkout main
git pull origin main

# Create a release branch (optional, for final prep)
git checkout -b release/v1.2.3
```

### Step 2: Update Version Information

Update version in:
- `README.md` (if version is referenced)
- Any version constants in code
- `docker-compose.prod.yml` image tags (if hardcoded)

### Step 3: Update CHANGELOG.md

Add a new section for the release:

```markdown
## [1.2.3] - 2025-12-29

### Added
- New email filtering API endpoint
- WebSocket support for real-time notifications

### Changed
- Improved database query performance by 40%
- Updated PostgreSQL driver to v1.6.0

### Fixed
- Fixed memory leak in email parsing
- Resolved race condition in WebSocket handler

### Security
- Updated dependencies with security patches
- Fixed potential SQL injection in search endpoint
```

### Step 4: Commit and Push Changes

```bash
# Commit version changes
git add .
git commit -m "chore: prepare release v1.2.3"

# Push to origin
git push origin release/v1.2.3
```

### Step 5: Create Pull Request

1. Create PR from `release/v1.2.3` to `main`
2. Title: "Release v1.2.3"
3. Description: Include changelog excerpt
4. Wait for all quality gates to pass
5. Get required approvals
6. Merge the PR

### Step 6: Create Release Tag

```bash
# After PR is merged, update main
git checkout main
git pull origin main

# Create annotated tag
git tag -a v1.2.3 -m "Release version 1.2.3"

# Push tag to trigger release workflow
git push origin v1.2.3
```

### Step 7: Monitor Release Workflow

1. Navigate to GitHub Actions: `https://github.com/<org>/<repo>/actions`
2. Find the "Release" workflow for your tag
3. Monitor each job:
   - Validate Release Tag
   - Run Full Test Suite
   - Build & Push Docker Image
   - Generate Changelog
   - Create GitHub Release

### Step 8: Verify Release

After workflow completes:

1. Check GitHub Releases page
2. Verify release notes are correct
3. Verify Docker image was pushed
4. Test Docker image pull:

```bash
docker pull ghcr.io/<org>/<repo>:1.2.3
```

## Automated Release Workflow

Our release workflow (`.github/workflows/release.yml`) automatically:

1. **Validates the tag** - Ensures semantic version format
2. **Runs all tests** - Unit, integration, and E2E tests
3. **Runs linter** - Ensures code quality
4. **Builds Docker image** - Multi-platform (amd64, arm64)
5. **Pushes to registry** - GitHub Container Registry (ghcr.io)
6. **Generates changelog** - From conventional commits
7. **Creates GitHub release** - With release notes and assets
8. **Tags Docker image** - With version and 'latest' (for stable)

### Workflow Triggers

The workflow triggers on tags matching: `v*.*.*`

Examples:
- `v1.2.3` - Triggers release workflow
- `v1.3.0-beta.1` - Triggers release workflow (pre-release)
- `release-1.2.3` - Does NOT trigger
- `1.2.3` - Does NOT trigger (missing 'v' prefix)

### Docker Image Tags

For stable release `v1.2.3`, images are tagged:
- `ghcr.io/<org>/<repo>:1.2.3`
- `ghcr.io/<org>/<repo>:1.2`
- `ghcr.io/<org>/<repo>:1`
- `ghcr.io/<org>/<repo>:latest`

For pre-release `v1.3.0-beta.1`, images are tagged:
- `ghcr.io/<org>/<repo>:1.3.0-beta.1`
- `ghcr.io/<org>/<repo>:1.3.0-beta`
- No 'latest' tag for pre-releases

## Post-Release Verification

### Immediate Checks (Within 1 hour)

- [ ] GitHub Release created successfully
- [ ] Docker image available in registry
- [ ] Image tags are correct (version, latest)
- [ ] Release notes are accurate
- [ ] No broken links in release notes

### Smoke Tests (Within 4 hours)

Test the released Docker image:

```bash
# Pull the release image
docker pull ghcr.io/<org>/<repo>:1.2.3

# Run smoke test using docker-compose
docker-compose -f docker-compose.prod.yml up -d

# Verify health endpoint
curl http://localhost:8080/health

# Check logs for errors
docker-compose -f docker-compose.prod.yml logs backend

# Run API smoke tests
make test-api
```

### Production Deployment (Within 24 hours)

- [ ] Deploy to staging environment
- [ ] Run full test suite against staging
- [ ] Verify all integrations working
- [ ] Check monitoring/metrics
- [ ] Deploy to production (if staging passes)
- [ ] Monitor production for 24 hours

### Monitoring (First 24 hours)

Watch for:
- Error rates in logs
- Performance degradation
- Memory leaks
- Database connection issues
- API error responses
- User-reported issues

## Rollback Procedure

If critical issues are discovered:

### Quick Rollback (Docker)

```bash
# Revert to previous version
docker-compose down
docker pull ghcr.io/<org>/<repo>:1.2.2  # Previous stable version
docker-compose up -d
```

### Database Rollback

If migration was included:

```bash
# Run rollback migration
make migrate-down

# Or use manual SQL script
psql $DATABASE_URL < migrations/rollback_v1.2.3.sql
```

### Tag Rollback (GitHub)

If release was published but is broken:

1. **Mark as Pre-release:**
   - Edit the GitHub Release
   - Check "This is a pre-release"
   - Add warning to description

2. **Create Hotfix Release:**
   ```bash
   # Create hotfix branch from previous version
   git checkout v1.2.2
   git checkout -b hotfix/v1.2.3-fix

   # Apply fixes
   git commit -m "fix: critical issue from v1.2.3"

   # Create new patch version
   git tag -a v1.2.4 -m "Hotfix for v1.2.3"
   git push origin v1.2.4
   ```

3. **Delete Bad Tag (Last Resort):**
   ```bash
   # Delete local tag
   git tag -d v1.2.3

   # Delete remote tag
   git push --delete origin v1.2.3

   # Delete GitHub Release manually
   ```

**NOTE:** Only delete tags as a last resort. Prefer marking as pre-release or creating a hotfix.

## Troubleshooting

### Release Workflow Failed

**Problem:** Release workflow fails on tests

**Solution:**
1. Check workflow logs for specific failure
2. Fix issues locally
3. Delete the tag: `git push --delete origin v1.2.3`
4. Fix the code, create new PR
5. Create new tag after fixes merged

**Problem:** Docker build fails

**Solution:**
1. Verify Dockerfile syntax locally: `docker build .`
2. Check for missing files in .dockerignore
3. Verify base image is accessible
4. Re-trigger workflow if transient failure

**Problem:** Changelog generation fails

**Solution:**
1. Ensure git history is available (fetch-depth: 0)
2. Check commit message format
3. Verify previous tag exists
4. Review changelog generation script

### Docker Image Issues

**Problem:** Image not appearing in registry

**Solution:**
1. Verify GITHUB_TOKEN has packages:write permission
2. Check registry login succeeded
3. Verify push step completed
4. Check GitHub Packages page

**Problem:** Wrong tags applied

**Solution:**
1. Review docker/metadata-action configuration
2. Verify tag format matches semver pattern
3. Check is-prerelease detection logic
4. Manually tag if needed:
   ```bash
   docker tag ghcr.io/<org>/<repo>:1.2.3 ghcr.io/<org>/<repo>:latest
   docker push ghcr.io/<org>/<repo>:latest
   ```

### Version Conflicts

**Problem:** Version already exists

**Solution:**
1. Check if tag already exists: `git tag -l "v1.2.3"`
2. Delete existing tag if incorrect:
   ```bash
   git tag -d v1.2.3
   git push --delete origin v1.2.3
   ```
3. Increment to next version instead

**Problem:** Version number wrong

**Solution:**
1. Delete tag before release is created
2. Fix version in code/CHANGELOG
3. Create new tag with correct version

## Best Practices

### Commit Message Format

Use conventional commits for automatic changelog generation:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature (MINOR version)
- `fix`: Bug fix (PATCH version)
- `docs`: Documentation only
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance tasks

**Examples:**
```
feat(api): add email filtering endpoint

Implements a new endpoint /api/emails/filter that allows
filtering emails by sender, subject, and date range.

Closes #123
```

```
fix(websocket): resolve race condition in handler

The WebSocket handler had a race condition when multiple
clients connected simultaneously. This fix adds proper
locking to prevent concurrent map writes.

Fixes #456
```

### Release Cadence

**Recommended schedule:**
- **Patch releases:** As needed (bug fixes)
- **Minor releases:** Every 2-4 weeks (new features)
- **Major releases:** Every 3-6 months (breaking changes)

**Exception:** Security patches should be released immediately.

### Communication

**Before release:**
- Announce planned release in team chat
- Coordinate with frontend team (API changes)
- Notify DevOps team (infrastructure changes)

**After release:**
- Announce in team chat with release notes link
- Update deployment documentation
- Notify stakeholders of new features

### Testing Pre-Releases

Always test pre-releases in non-production environments:

1. **Alpha** - Internal dev testing
2. **Beta** - Staging environment testing
3. **RC** - Production-like environment testing
4. **Stable** - Production deployment

## Security Considerations

### Secrets Management

- Never commit secrets to repository
- Use GitHub Secrets for CI/CD credentials
- Rotate CODECOV_TOKEN and GITHUB_TOKEN regularly
- Audit access to repository secrets

### Dependency Updates

- Review dependency changes in each release
- Check for known vulnerabilities: `go list -m all | nancy sleuth`
- Update vulnerable dependencies before release
- Document dependency changes in CHANGELOG

### Container Security

- Use minimal base images (alpine)
- Scan images for vulnerabilities
- Sign container images (optional)
- Use specific version tags, not 'latest' in production

## Release Artifacts

Each release includes:

1. **GitHub Release** - Release notes, changelog, links
2. **Docker Images** - Multi-platform container images
3. **Git Tag** - Annotated tag with version info
4. **CHANGELOG.md** - Updated with release notes

## Maintenance

Review this guide:
- After each major release
- When release process changes
- When tooling updates
- Quarterly for best practices

## References

- [Semantic Versioning](https://semver.org/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Keep a Changelog](https://keepachangelog.com/)
- [GitHub Releases](https://docs.github.com/en/repositories/releasing-projects-on-github)

## Support

For questions or issues with the release process:

1. Check this guide first
2. Review recent release workflow runs
3. Consult with ATLAS (Team Lead)
4. Create issue in repository

---

**Last Updated:** 2025-12-29
**Owner:** VALIDATOR (QA & Release Engineer)
**Catchphrase:** Ship with confidence
