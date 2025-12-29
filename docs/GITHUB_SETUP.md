# GitHub Repository Setup Guide

This guide walks through setting up GitHub repository settings, secrets, and integrations for the Infinimail backend CI/CD pipeline.

## Table of Contents

- [Repository Secrets](#repository-secrets)
- [Branch Protection Rules](#branch-protection-rules)
- [GitHub Actions Permissions](#github-actions-permissions)
- [Codecov Integration](#codecov-integration)
- [GitHub Container Registry](#github-container-registry)
- [Verification](#verification)

## Repository Secrets

Navigate to: **Settings > Secrets and variables > Actions**

### Required Secrets

#### 1. CODECOV_TOKEN

**Purpose:** Upload coverage reports to Codecov

**How to get:**
1. Sign up at https://app.codecov.io/
2. Add your GitHub repository
3. Copy the upload token
4. Add as repository secret

**Steps:**
```
1. Go to https://app.codecov.io/
2. Click "Add new repository"
3. Select your repository
4. Copy the "Repository Upload Token"
5. In GitHub: Settings > Secrets and variables > Actions > New repository secret
   Name: CODECOV_TOKEN
   Secret: [paste token]
```

**Verification:**
```bash
# In workflow, the token is used like:
uses: codecov/codecov-action@v4
with:
  token: ${{ secrets.CODECOV_TOKEN }}
```

#### 2. GITHUB_TOKEN (Auto-created)

**Purpose:** GitHub Actions default token for releases and packages

**Setup:** Automatically available in workflows, no setup needed

**Permissions:** Configured in workflow files (see below)

**Usage:**
- Create GitHub Releases
- Push to GitHub Container Registry (ghcr.io)
- Post comments on PRs
- Create/update checks

### Optional Secrets

#### 3. SLACK_WEBHOOK_URL (Future)

**Purpose:** Send release notifications to Slack

**How to get:**
1. Create Slack app
2. Enable Incoming Webhooks
3. Copy webhook URL

#### 4. SENTRY_DSN (Future)

**Purpose:** Error tracking in production releases

## GitHub Actions Permissions

### Workflow Permissions

Navigate to: **Settings > Actions > General > Workflow permissions**

**Required settings:**
```
[x] Read and write permissions
    Workflows can read and write to the repository

[x] Allow GitHub Actions to create and approve pull requests
    (For automated dependency updates - optional)
```

**Why needed:**
- Read: Clone repository, read code
- Write: Create releases, push Docker images, update tags

### Per-Workflow Permissions

Workflows define their own permissions in YAML:

**test.yml:**
```yaml
permissions:
  contents: read
  checks: write
  pull-requests: write
```

**release.yml:**
```yaml
permissions:
  contents: write
  packages: write
  pull-requests: read
```

## Branch Protection Rules

Navigate to: **Settings > Branches > Branch protection rules**

See detailed setup in: [BRANCH_PROTECTION.md](./BRANCH_PROTECTION.md)

### Quick Setup

1. Click "Add rule"
2. Branch name pattern: `main`
3. Enable:
   - [x] Require a pull request before merging
   - [x] Require approvals: 1
   - [x] Require status checks to pass before merging
   - [x] Require branches to be up to date before merging
4. Add required status checks:
   - quality-gate
   - lint
   - unit-tests
   - integration-tests
   - e2e-tests
   - coverage-report
   - codecov/project
   - codecov/patch
5. Enable:
   - [x] Do not allow force pushes
   - [x] Do not allow deletions
6. Click "Create"

## Codecov Integration

### Setup Steps

1. **Sign up for Codecov**
   - Visit: https://app.codecov.io/
   - Sign in with GitHub
   - Authorize Codecov app

2. **Add Repository**
   - Click "Add new repository"
   - Select `webrana-infinimail-backend`
   - Copy the upload token

3. **Configure Repository**
   - Token added to GitHub secrets (see above)
   - `codecov.yml` already in repository root

4. **Verify Integration**
   - Push a commit to trigger CI
   - Check Codecov dashboard after workflow completes
   - Verify coverage badge (optional)

### Codecov Configuration

File: `codecov.yml` (already created)

**Key settings:**
- Project coverage threshold: 70%
- Patch coverage threshold: 70%
- Individual flag targets (unit: 75%, integration: 65%, e2e: 60%)

### Coverage Badge (Optional)

Add to README.md:

```markdown
[![codecov](https://codecov.io/gh/<org>/<repo>/branch/main/graph/badge.svg)](https://codecov.io/gh/<org>/<repo>)
```

Get badge URL from Codecov settings.

## GitHub Container Registry

### Enable Container Registry

GitHub Container Registry (ghcr.io) is automatically available.

### Configure Package Visibility

1. Navigate to: **Packages** (on repository main page)
2. After first image is pushed, find package
3. Click "Package settings"
4. Set visibility:
   - Public (recommended for open source)
   - Private (recommended for proprietary)

### Link Package to Repository

1. In package settings
2. Click "Connect repository"
3. Select your repository
4. This adds the package to your repo page

### Pull Image

After release, pull image:

```bash
# Public package
docker pull ghcr.io/<org>/<repo>:latest

# Private package (requires authentication)
echo $GITHUB_TOKEN | docker login ghcr.io -u <username> --password-stdin
docker pull ghcr.io/<org>/<repo>:latest
```

### Image Permissions

Package inherits repository permissions by default.

## Environment Setup (Optional)

For production deployments, create environments:

Navigate to: **Settings > Environments**

### Create Production Environment

1. Click "New environment"
2. Name: `production`
3. Configure:
   - [x] Required reviewers: 1
   - [x] Wait timer: 0 minutes
   - Deployment branches: `main` only
4. Add environment secrets if needed

### Environment Secrets

Different from repository secrets:
- Scoped to specific environment
- Only accessible in jobs targeting that environment

Example: Production API keys, database URLs

## Repository Settings

### General Settings

Navigate to: **Settings > General**

**Recommended settings:**

1. **Features:**
   - [x] Issues
   - [x] Projects (optional)
   - [x] Wiki (optional)
   - [x] Discussions (optional)

2. **Pull Requests:**
   - [x] Allow squash merging
   - [x] Allow rebase merging
   - [ ] Allow merge commits (disable for linear history)
   - [x] Automatically delete head branches

3. **Archives:**
   - [x] Include Git LFS objects in archives

### Actions Settings

Navigate to: **Settings > Actions > General**

1. **Actions permissions:**
   - [x] Allow all actions and reusable workflows

2. **Fork pull request workflows:**
   - [ ] Run workflows from fork pull requests (disabled for security)

3. **Workflow permissions:**
   - [x] Read and write permissions

## Webhooks (Optional)

### Setup Deployment Webhook

For automatic deployment notifications:

Navigate to: **Settings > Webhooks > Add webhook**

1. Payload URL: `https://your-deploy-service.com/webhook`
2. Content type: `application/json`
3. Events:
   - [x] Releases
   - [x] Workflow runs
4. Click "Add webhook"

## Verification Checklist

After completing setup, verify:

### Secrets
- [ ] CODECOV_TOKEN is set
- [ ] GITHUB_TOKEN permissions are correct

### Branch Protection
- [ ] Main branch is protected
- [ ] Required status checks are configured
- [ ] Required reviewers are set
- [ ] Force push is disabled

### Codecov
- [ ] Repository added to Codecov
- [ ] Upload token configured
- [ ] codecov.yml is in repository
- [ ] Coverage badge displays (optional)

### GitHub Actions
- [ ] Workflow permissions configured
- [ ] Test workflow runs successfully
- [ ] Release workflow ready (test with pre-release tag)

### Container Registry
- [ ] Package visibility set
- [ ] Package linked to repository
- [ ] Can pull images (test with: `docker pull ghcr.io/<org>/<repo>:latest`)

## Testing the Setup

### Test CI Pipeline

1. Create a feature branch:
   ```bash
   git checkout -b test/ci-setup
   ```

2. Make a small change:
   ```bash
   echo "# CI Test" >> README.md
   git add README.md
   git commit -m "test: verify CI pipeline"
   git push origin test/ci-setup
   ```

3. Create Pull Request

4. Verify:
   - [ ] All status checks run
   - [ ] Lint check passes
   - [ ] Unit tests pass
   - [ ] Integration tests pass
   - [ ] E2E tests pass
   - [ ] Coverage uploaded to Codecov
   - [ ] Quality gate check passes

5. Check Codecov:
   - Visit Codecov dashboard
   - Verify coverage report is visible
   - Check PR comment (if enabled)

### Test Release Pipeline

1. Create a test pre-release:
   ```bash
   git checkout main
   git pull origin main
   git tag -a v0.0.1-test.1 -m "Test release"
   git push origin v0.0.1-test.1
   ```

2. Monitor release workflow:
   - Visit Actions tab
   - Watch "Release" workflow
   - Verify all jobs pass

3. Check outputs:
   - [ ] GitHub Release created
   - [ ] Release notes generated
   - [ ] Docker image pushed to ghcr.io
   - [ ] Image tagged correctly

4. Pull test image:
   ```bash
   docker pull ghcr.io/<org>/<repo>:0.0.1-test.1
   docker run --rm ghcr.io/<org>/<repo>:0.0.1-test.1 --version
   ```

5. Clean up:
   ```bash
   # Delete test tag
   git tag -d v0.0.1-test.1
   git push --delete origin v0.0.1-test.1

   # Delete test release from GitHub UI
   ```

## Troubleshooting

### Codecov Upload Fails

**Error:** `Error uploading coverage reports`

**Solutions:**
1. Verify CODECOV_TOKEN is set correctly
2. Check token hasn't expired
3. Verify repository is added to Codecov
4. Check Codecov service status

### Docker Push Fails

**Error:** `denied: permission_denied`

**Solutions:**
1. Verify workflow permissions include `packages: write`
2. Check GITHUB_TOKEN is being used
3. Verify package visibility settings
4. Check organization settings (if applicable)

### Status Checks Not Appearing

**Issue:** Required status checks not showing in PR

**Solutions:**
1. Ensure workflows ran at least once on branch
2. Check workflow names match exactly
3. Verify workflows are not disabled
4. Check if workflows are restricted by branch

### Branch Protection Too Strict

**Issue:** Can't merge even though all checks pass

**Solutions:**
1. Verify branch is up to date with main
2. Check all required checks are in the list
3. Ensure required approvals are met
4. Check if user has bypass permissions

## Maintenance

### Regular Tasks

**Weekly:**
- Review failed workflow runs
- Check Codecov trends

**Monthly:**
- Review and update required status checks
- Audit repository secrets
- Check for deprecated GitHub Actions

**Quarterly:**
- Review branch protection rules
- Update CI/CD pipeline (Actions versions)
- Review and update this guide

### Secret Rotation

Rotate secrets periodically:

1. **CODECOV_TOKEN:**
   - Generate new token in Codecov
   - Update in GitHub secrets
   - Test with a PR

2. **GITHUB_TOKEN:**
   - Automatically rotated by GitHub
   - No manual action needed

## References

- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [GitHub Secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [Branch Protection](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches)
- [Codecov Documentation](https://docs.codecov.com/)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)

## Support

For issues with setup:
1. Check this guide first
2. Review GitHub Actions logs
3. Check Codecov logs
4. Consult with ATLAS (Team Lead)

---

**Last Updated:** 2025-12-29
**Owner:** VALIDATOR (QA & Release Engineer)
