#!/bin/bash
# Pre-commit validation hook for SafeEdge
# This provides NON-BLOCKING hints to Claude about code quality issues

set -e

# Colors for output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

echo "🔍 Running pre-commit checks..."

# Track if any hints were found
HINTS_FOUND=0

# Check 1: Go formatting
echo "Checking Go formatting..."
UNFORMATTED=$(gofmt -l . 2>/dev/null || true)
if [ -n "$UNFORMATTED" ]; then
  echo -e "${YELLOW}HINT: Some files are not formatted. Run 'go fmt ./...'${NC}"
  echo "$UNFORMATTED"
  HINTS_FOUND=1
fi

# Check 2: sqlc regeneration needed
if git diff --cached --name-only | grep -q "internal/controlplane/database/\(schema.sql\|queries/\)"; then
  echo "Checking if sqlc code is up to date..."
  if [ -d "internal/controlplane/database" ]; then
    # Check if generated/ directory exists and has recent files
    if [ ! -d "internal/controlplane/database/generated" ]; then
      echo -e "${YELLOW}HINT: Database schema/queries changed but generated/ doesn't exist. Run 'cd internal/controlplane/database && sqlc generate'${NC}"
      HINTS_FOUND=1
    else
      SCHEMA_MOD=$(stat -f %m "internal/controlplane/database/schema.sql" 2>/dev/null || echo 0)
      GEN_MOD=$(find internal/controlplane/database/generated -type f -exec stat -f %m {} \; 2>/dev/null | sort -n | tail -1 || echo 0)
      if [ "$SCHEMA_MOD" -gt "$GEN_MOD" ]; then
        echo -e "${YELLOW}HINT: schema.sql modified more recently than generated code. Run 'cd internal/controlplane/database && sqlc generate'${NC}"
        HINTS_FOUND=1
      fi
    fi
  fi
fi

# Check 3: No secrets in staged files
echo "Checking for potential secrets..."
STAGED_FILES=$(git diff --cached --name-only)
for file in $STAGED_FILES; do
  if [ -f "$file" ]; then
    # Check for common secret patterns
    if grep -qE "(api_key|apikey|secret|password|token|private_key).*=.*['\"][\w]{20,}" "$file" 2>/dev/null; then
      echo -e "${YELLOW}HINT: File '$file' may contain secrets. Please review before committing.${NC}"
      HINTS_FOUND=1
    fi
  fi
done

# Check 4: Verify .env files are not staged
if echo "$STAGED_FILES" | grep -q "\.env$"; then
  echo -e "${RED}WARNING: .env file is staged. This typically contains secrets!${NC}"
  HINTS_FOUND=1
fi

# Summary
if [ $HINTS_FOUND -eq 0 ]; then
  echo -e "${GREEN}✓ All pre-commit checks passed!${NC}"
  exit 0
else
  echo ""
  echo -e "${YELLOW}ℹ️  Some hints were found. These are suggestions, not blockers.${NC}"
  echo -e "${YELLOW}   Review the hints above and fix if needed.${NC}"
  exit 0  # Non-blocking - always exit 0
fi
