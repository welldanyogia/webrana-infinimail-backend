#!/bin/bash

# Quality Gates & Release Setup Verification Script
# Verifies that all quality gate and release automation files are in place
# and properly configured.

set -e

echo "============================================"
echo "Quality Gates & Release Setup Verification"
echo "============================================"
echo ""

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
WARNINGS=0

# Helper functions
check_file() {
    if [ -f "$1" ]; then
        echo -e "${GREEN}✓${NC} $1 exists"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC} $1 missing"
        ((FAILED++))
        return 1
    fi
}

check_file_content() {
    if grep -q "$2" "$1" 2>/dev/null; then
        echo -e "${GREEN}✓${NC} $1 contains '$2'"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC} $1 missing '$2'"
        ((FAILED++))
        return 1
    fi
}

warn_message() {
    echo -e "${YELLOW}⚠${NC} $1"
    ((WARNINGS++))
}

section() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "$1"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Check GitHub Actions workflows
section "GitHub Actions Workflows"

check_file ".github/workflows/test.yml"
if [ -f ".github/workflows/test.yml" ]; then
    check_file_content ".github/workflows/test.yml" "quality-gate"
    check_file_content ".github/workflows/test.yml" "codecov"
    check_file_content ".github/workflows/test.yml" "COVERAGE_THRESHOLD"
fi

check_file ".github/workflows/release.yml"
if [ -f ".github/workflows/release.yml" ]; then
    check_file_content ".github/workflows/release.yml" "semver"
    check_file_content ".github/workflows/release.yml" "docker"
    check_file_content ".github/workflows/release.yml" "ghcr.io"
fi

# Check configuration files
section "Configuration Files"

check_file "codecov.yml"
if [ -f "codecov.yml" ]; then
    check_file_content "codecov.yml" "target: 70%"
    check_file_content "codecov.yml" "unit:"
    check_file_content "codecov.yml" "integration:"
fi

check_file "Makefile"
if [ -f "Makefile" ]; then
    check_file_content "Makefile" "quality-gate:"
    check_file_content "Makefile" "check-coverage:"
    check_file_content "Makefile" "test-coverage"
fi

# Check documentation
section "Documentation Files"

check_file "docs/QUALITY_GATES.md"
check_file "docs/RELEASE.md"
check_file "docs/BRANCH_PROTECTION.md"
check_file "docs/GITHUB_SETUP.md"
check_file "docs/QUICK_REFERENCE.md"
check_file "QA_RELEASE_SETUP.md"

# Check test infrastructure
section "Test Infrastructure"

if [ -d "tests" ]; then
    echo -e "${GREEN}✓${NC} tests/ directory exists"
    ((PASSED++))

    if [ -d "tests/integration" ]; then
        echo -e "${GREEN}✓${NC} tests/integration/ directory exists"
        ((PASSED++))
    fi

    if [ -d "tests/e2e" ]; then
        echo -e "${GREEN}✓${NC} tests/e2e/ directory exists"
        ((PASSED++))
    fi
else
    echo -e "${RED}✗${NC} tests/ directory missing"
    ((FAILED++))
fi

# Check Dockerfile
section "Docker Configuration"

check_file "Dockerfile"
check_file "docker-compose.yml"
check_file "docker-compose.prod.yml"

# Check Go configuration
section "Go Configuration"

check_file "go.mod"
if [ -f "go.mod" ]; then
    check_file_content "go.mod" "go 1.24"
fi

# Check for testing dependencies
if [ -f "go.mod" ]; then
    if grep -q "testcontainers" go.mod; then
        echo -e "${GREEN}✓${NC} testcontainers dependency present"
        ((PASSED++))
    fi

    if grep -q "testify" go.mod; then
        echo -e "${GREEN}✓${NC} testify dependency present"
        ((PASSED++))
    fi
fi

# Verify tools are installed
section "Development Tools"

if command -v golangci-lint &> /dev/null; then
    echo -e "${GREEN}✓${NC} golangci-lint is installed"
    ((PASSED++))
else
    warn_message "golangci-lint not installed (run: make install-tools)"
fi

if command -v go &> /dev/null; then
    echo -e "${GREEN}✓${NC} Go is installed"
    GO_VERSION=$(go version | awk '{print $3}')
    echo "  Version: $GO_VERSION"
    ((PASSED++))

    if [[ "$GO_VERSION" == *"1.24"* ]]; then
        echo -e "${GREEN}✓${NC} Go version matches CI (1.24)"
        ((PASSED++))
    else
        warn_message "Go version mismatch (CI uses 1.24, you have $GO_VERSION)"
    fi
else
    echo -e "${RED}✗${NC} Go not installed"
    ((FAILED++))
fi

if command -v docker &> /dev/null; then
    echo -e "${GREEN}✓${NC} Docker is installed"
    ((PASSED++))
else
    warn_message "Docker not installed (needed for integration tests)"
fi

# Check Makefile targets
section "Makefile Targets"

if [ -f "Makefile" ]; then
    REQUIRED_TARGETS=(
        "test"
        "test-unit"
        "test-integration"
        "test-e2e"
        "test-coverage"
        "test-race"
        "lint"
        "fmt"
        "check"
        "quality-gate"
        "check-coverage"
    )

    for target in "${REQUIRED_TARGETS[@]}"; do
        if grep -q "^${target}:" Makefile; then
            echo -e "${GREEN}✓${NC} Makefile target '${target}' exists"
            ((PASSED++))
        else
            echo -e "${RED}✗${NC} Makefile target '${target}' missing"
            ((FAILED++))
        fi
    done
fi

# Check environment setup
section "Environment Configuration"

if [ -f ".env.example" ]; then
    echo -e "${GREEN}✓${NC} .env.example exists"
    ((PASSED++))
else
    warn_message ".env.example not found"
fi

if [ -f ".env.secure.example" ]; then
    echo -e "${GREEN}✓${NC} .env.secure.example exists"
    ((PASSED++))
fi

# Check gitignore
if [ -f ".gitignore" ]; then
    if grep -q "coverage.out" .gitignore; then
        echo -e "${GREEN}✓${NC} .gitignore includes coverage.out"
        ((PASSED++))
    fi

    if grep -q "coverage.html" .gitignore; then
        echo -e "${GREEN}✓${NC} .gitignore includes coverage.html"
        ((PASSED++))
    fi
fi

# Final summary
section "Verification Summary"

echo ""
echo -e "Passed:   ${GREEN}${PASSED}${NC}"
echo -e "Failed:   ${RED}${FAILED}${NC}"
echo -e "Warnings: ${YELLOW}${WARNINGS}${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  Quality Gates Setup: COMPLETE ✓${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "Next steps:"
    echo "1. Configure GitHub secrets (see docs/GITHUB_SETUP.md)"
    echo "2. Apply branch protection rules (see docs/BRANCH_PROTECTION.md)"
    echo "3. Test CI pipeline with a PR"
    echo "4. Test release workflow with a tag"
    echo ""
    echo "Ship with confidence!"
    exit 0
else
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}  Quality Gates Setup: INCOMPLETE${NC}"
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "Please address the failed checks above."
    echo "See QA_RELEASE_SETUP.md for details."
    exit 1
fi
