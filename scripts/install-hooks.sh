#!/bin/bash
# Install git hooks for security scanning
# Run: ./scripts/install-hooks.sh

set -e

echo "========================================"
echo "🔧 Installing Git Hooks"
echo "========================================"

# Check if we're in a git repository
if [ ! -d .git ]; then
    echo "❌ Error: Not in a git repository"
    echo "   Run this script from the repository root"
    exit 1
fi

# Create .git/hooks directory if it doesn't exist
mkdir -p .git/hooks

# Install pre-commit hook
if [ -f scripts/pre-commit.sh ]; then
    echo "📋 Installing pre-commit hook..."
    cp scripts/pre-commit.sh .git/hooks/pre-commit
    chmod +x .git/hooks/pre-commit
    echo "✅ Pre-commit hook installed"
else
    echo "❌ Error: scripts/pre-commit.sh not found"
    exit 1
fi

echo ""
echo "========================================"
echo "✅ Git hooks installed successfully!"
echo "========================================"
echo ""
echo "The following hooks are now active:"
echo "  - pre-commit: Security scanning before commits"
echo ""
echo "What happens on commit:"
echo "  1. Gitleaks scans for secrets"
echo "  2. gosec scans for Go security issues"
echo "  3. gofmt checks code formatting"
echo "  4. go vet checks for suspicious code"
echo "  5. golangci-lint runs comprehensive checks"
echo ""
echo "To bypass hooks (NOT recommended):"
echo "  git commit --no-verify"
echo ""
echo "To uninstall hooks:"
echo "  rm .git/hooks/pre-commit"
echo ""
