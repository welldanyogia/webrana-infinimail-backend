# Branch Protection Rules Configuration

This document describes the required branch protection rules for the Infinimail backend repository to enforce quality gates.

## Overview

Branch protection rules ensure that all code merged into the main branch passes quality checks, maintains test coverage, and has been reviewed by team members.

## Required Configuration

### GitHub Repository Settings

Navigate to: **Settings > Branches > Branch protection rules**

### Main Branch Protection Rule

Create a rule for branch pattern: `main`

## Required Status Checks

Enable: **Require status checks to pass before merging**

### Status Checks to Require

The following checks MUST pass before any PR can be merged:

1. **Quality Gate Status** - Overall quality gate check
2. **Lint** - Code style and static analysis
3. **Unit Tests** - Fast unit tests with race detection
4. **Integration Tests** - Database integration tests
5. **E2E Tests** - End-to-end functionality tests
6. **Coverage Report & Quality Gate** - Coverage threshold enforcement
7. **codecov/project** - Codecov project coverage check
8. **codecov/patch** - Codecov patch coverage check

### Configuration Settings

```
[x] Require status checks to pass before merging
    [x] Require branches to be up to date before merging

    Required status checks:
    - quality-gate
    - lint
    - unit-tests
    - integration-tests
    - e2e-tests
    - coverage-report
    - codecov/project
    - codecov/patch
```

## Pull Request Reviews

Enable: **Require pull request reviews before merging**

### Review Settings

```
[x] Require a pull request before merging
    [x] Require approvals: 1
    [ ] Dismiss stale pull request approvals when new commits are pushed
    [x] Require review from Code Owners (if CODEOWNERS file exists)
    [ ] Restrict who can dismiss pull request reviews
    [x] Allow specified actors to bypass required pull requests
        - Repository administrators (for emergency fixes only)
```

## Commit Signing

Optional but recommended:

```
[x] Require signed commits
```

## Additional Protections

### Restrict Who Can Push

```
[x] Restrict pushes that create matching branches
    [ ] Restrict pushes
    ✓ Only allow specific people, teams, or apps to push
```

**Allowed actors:**
- Repository administrators (emergency fixes only)
- CI/CD service accounts (automated releases)

### Force Push Protection

```
[x] Do not allow force pushes
[x] Do not allow deletions
```

### Linear History

```
[ ] Require linear history (optional - enables squash or rebase only)
```

## Develop Branch Protection (Optional)

For teams using a develop/staging branch:

Create a rule for branch pattern: `develop`

```
[x] Require status checks to pass before merging
    [x] Require branches to be up to date before merging

    Required status checks:
    - lint
    - unit-tests
    - integration-tests

[x] Require pull request reviews before merging
    [x] Require approvals: 1
```

Less strict than main, but still maintains quality.

## Applying These Rules

### Step-by-Step Instructions

1. **Navigate to Branch Protection**
   - Go to repository **Settings**
   - Click **Branches** in the left sidebar
   - Click **Add rule** or **Add branch protection rule**

2. **Configure Branch Pattern**
   - Branch name pattern: `main`
   - Check **Require a pull request before merging**

3. **Enable Status Checks**
   - Check **Require status checks to pass before merging**
   - Check **Require branches to be up to date before merging**
   - In the search box, type each required check name and select it:
     - `quality-gate`
     - `lint`
     - `unit-tests`
     - `integration-tests`
     - `e2e-tests`
     - `coverage-report`
     - `codecov/project`
     - `codecov/patch`

4. **Configure Reviews**
   - Check **Require approvals**
   - Set number of required approvals: `1`

5. **Enable Force Push Protection**
   - Check **Do not allow force pushes**
   - Check **Do not allow deletions**

6. **Save Changes**
   - Click **Create** or **Save changes**

## Verification

After applying the rules, verify by:

1. Creating a test branch
2. Making a small change
3. Opening a pull request
4. Confirming that:
   - All status checks run
   - Merge button is disabled until checks pass
   - At least 1 approval is required

## Bypass Procedures

### Emergency Hotfixes

In case of critical production issues:

1. Repository administrators can bypass protections
2. Document the bypass in the PR description
3. Create a follow-up issue for proper testing
4. Conduct post-incident review

### Automated Releases

The release workflow can push tags without creating PRs:

- Tags matching `v*.*.*` trigger the release workflow
- Workflow has write permissions for packages and releases
- All quality gates run before release is created

## Rulesets (Alternative Approach)

GitHub now offers Repository Rulesets as an alternative to branch protection rules. To use rulesets:

1. Navigate to **Settings > Rules > Rulesets**
2. Click **New ruleset > New branch ruleset**
3. Configure similar protections with more granular control

Rulesets offer:
- More flexible targeting (multiple branches, patterns)
- Better audit logging
- Bypass permissions per rule
- Required workflows

## Quality Gate Criteria

A PR can only be merged when ALL of the following are true:

| Check | Requirement | Enforced By |
|-------|------------|-------------|
| Linting | No linting errors | golangci-lint workflow |
| Unit Tests | All pass with race detection | unit-tests workflow |
| Integration Tests | All pass with real database | integration-tests workflow |
| E2E Tests | All pass end-to-end | e2e-tests workflow |
| Code Coverage | Project: >= 70%, Patch: >= 70% | codecov + coverage-report |
| Quality Gate | All jobs successful | quality-gate workflow |
| PR Review | >= 1 approval | GitHub settings |
| Up to Date | Branch synced with main | GitHub settings |

## Monitoring and Alerts

### GitHub Actions Dashboard

Monitor workflow runs at: `https://github.com/<org>/<repo>/actions`

### Codecov Dashboard

View coverage trends at: `https://app.codecov.io/gh/<org>/<repo>`

### Failed Checks

When checks fail:

1. Click on the failed check in the PR
2. Review the workflow logs
3. Fix the issues locally
4. Push new commits
5. Checks will re-run automatically

## Best Practices

1. **Never Skip Checks** - Quality gates exist for a reason
2. **Fix Don't Override** - Fix issues rather than bypassing rules
3. **Review Coverage** - Don't just meet threshold, write meaningful tests
4. **Update Rules** - Review and adjust rules quarterly
5. **Document Bypasses** - Always document why you bypassed a rule
6. **Monitor Trends** - Track coverage and test stability over time

## Troubleshooting

### Status Checks Not Appearing

- Ensure the workflow has run at least once on the branch
- Check that workflow names match exactly
- Verify workflows are enabled in repository settings

### Codecov Checks Failing

- Ensure `CODECOV_TOKEN` secret is configured
- Check codecov.yml configuration
- Verify coverage files are being generated

### Can't Merge Despite Passing Checks

- Ensure branch is up to date with main
- Check if reviews are required and approved
- Verify all required checks are listed

## References

- [GitHub Branch Protection Documentation](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)
- [GitHub Actions Required Checks](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/collaborating-on-repositories-with-code-quality-features/about-status-checks)
- [Codecov GitHub Integration](https://docs.codecov.com/docs/github-integration)

## Maintenance

This configuration should be reviewed:

- Quarterly by the development team
- After major workflow changes
- When introducing new test types
- Following security incidents

**Last Updated:** 2025-12-29
**Owner:** VALIDATOR (QA & Release Engineer)
