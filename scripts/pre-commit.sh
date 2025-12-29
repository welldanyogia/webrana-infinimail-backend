#!/bin/bash
# Pre-commit hook for security scanning
# Install: cp scripts/pre-commit.sh .git/hooks/pre-commit && chmod +x .git/hooks/pre-commit

set -e

echo "========================================"
echo "🔒 Running pre-commit security checks"
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track if any checks fail
CHECKS_FAILED=0

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2${NC}"
    else
        echo -e "${RED}❌ $2${NC}"
        CHECKS_FAILED=1
    fi
}

# 1. Check for secrets with Gitleaks
echo ""
echo "🔍 Step 1/5: Scanning for secrets..."
if command -v gitleaks &> /dev/null; then
    if gitleaks protect --staged -v --no-banner 2>&1 | grep -q "Finding:"; then
        echo -e "${RED}❌ Gitleaks found potential secrets!${NC}"
        echo -e "${YELLOW}⚠️  COMMIT BLOCKED - Remove secrets before committing${NC}"
        exit 1
    else
        print_status 0 "No secrets found"
    fi
else
    echo -e "${YELLOW}⚠️  Gitleaks not installed - skipping secret scan${NC}"
    echo "   Install: brew install gitleaks (macOS) or see https://github.com/gitleaks/gitleaks"
fi

# 2. Run gosec on changed Go files
echo ""
echo "🔍 Step 2/5: Running gosec security scanner..."
if command -v gosec &> /dev/null; then
    # Get list of changed Go files
    CHANGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

    if [ -n "$CHANGED_GO_FILES" ]; then
        if gosec -exclude-dir=tests ./... > /dev/null 2>&1; then
            print_status 0 "gosec found no security issues"
        else
            echo -e "${YELLOW}⚠️  gosec found potential security issues${NC}"
            echo "   Run: gosec ./... for details"
            echo "   (Not blocking commit, but review recommended)"
        fi
    else
        echo "   No Go files changed - skipping"
    fi
else
    echo -e "${YELLOW}⚠️  gosec not installed - skipping Go security scan${NC}"
    echo "   Install: go install github.com/securego/gosec/v2/cmd/gosec@latest"
fi

# 3. Run gofmt check
echo ""
echo "🔍 Step 3/5: Checking Go formatting..."
CHANGED_GO_FILES=$(git diff --cached --name-only --diff-filter=ACM | grep '\.go$' || true)

if [ -n "$CHANGED_GO_FILES" ]; then
    UNFORMATTED_FILES=$(echo "$CHANGED_GO_FILES" | xargs gofmt -l || true)

    if [ -n "$UNFORMATTED_FILES" ]; then
        echo -e "${YELLOW}⚠️  The following files need formatting:${NC}"
        echo "$UNFORMATTED_FILES"
        echo "   Run: gofmt -w $UNFORMATTED_FILES"
        echo "   Or:  make fmt"
        CHECKS_FAILED=1
    else
        print_status 0 "All Go files properly formatted"
    fi
else
    echo "   No Go files changed - skipping"
fi

# 4. Run go vet
echo ""
echo "🔍 Step 4/5: Running go vet..."
if [ -n "$CHANGED_GO_FILES" ]; then
    if go vet ./... > /dev/null 2>&1; then
        print_status 0 "go vet found no issues"
    else
        echo -e "${YELLOW}⚠️  go vet found potential issues${NC}"
        echo "   Run: go vet ./... for details"
        CHECKS_FAILED=1
    fi
else
    echo "   No Go files changed - skipping"
fi

# 5. Run golangci-lint (if available)
echo ""
echo "🔍 Step 5/5: Running golangci-lint..."
if command -v golangci-lint &> /dev/null; then
    if [ -n "$CHANGED_GO_FILES" ]; then
        if golangci-lint run --new-from-rev=HEAD~1 --config=.golangci.yml > /dev/null 2>&1; then
            print_status 0 "golangci-lint found no issues"
        else
            echo -e "${YELLOW}⚠️  golangci-lint found issues in new code${NC}"
            echo "   Run: golangci-lint run for details"
            echo "   (Not blocking commit, but review recommended)"
        fi
    else
        echo "   No Go files changed - skipping"
    fi
else
    echo -e "${YELLOW}⚠️  golangci-lint not installed - skipping comprehensive linting${NC}"
    echo "   Install: https://golangci-lint.run/usage/install/"
fi

# Final summary
echo ""
echo "========================================"
if [ $CHECKS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All pre-commit checks passed!${NC}"
    echo "========================================"
    exit 0
else
    echo -e "${RED}❌ Some checks failed - please fix before committing${NC}"
    echo "========================================"
    echo ""
    echo "To bypass this hook (NOT recommended):"
    echo "  git commit --no-verify"
    echo ""
    exit 1
fi
