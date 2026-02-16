#!/bin/bash
# sync-skip-tests-deploy.sh — Sync, commit, push, release — WITHOUT running tests.
# Usage: ./sync-skip-tests-deploy.sh ["commit message"]
# Steps: sync files -> verify no leaks -> commit -> push -> trigger release
# Like sync-test-deploy.sh but skips all tests for speed.
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
DEV_REPO="$SCRIPT_DIR"
PUB_REPO="${PUB_REPO:-$(dirname "$DEV_REPO")/allyourbase}"
GH_REPO="gridlhq/allyourbase"

COMMIT_MESSAGE="${1:-Sync from dev repo}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}AYB Sync & Deploy (SKIP TESTS)${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Verify repos exist
if [ ! -d "$DEV_REPO" ]; then
  echo -e "${RED}Error: Dev repo not found at $DEV_REPO${NC}"
  exit 1
fi

if [ ! -d "$PUB_REPO" ]; then
  echo -e "${RED}Error: Public repo not found at $PUB_REPO${NC}"
  echo "Clone it first: git clone git@github.com:gridlhq/allyourbase.git $PUB_REPO"
  exit 1
fi

cd "$DEV_REPO"
if [ ! -f "sync-to-public.sh" ]; then
  echo -e "${RED}Error: Not in the correct dev repo directory${NC}"
  exit 1
fi

# Detect current branch from dev repo
CURRENT_BRANCH=$(git branch --show-current)
echo -e "${BLUE}Branch: ${GREEN}$CURRENT_BRANCH${NC}"
echo ""

# ── Step 0: Ensure public repo is on matching branch ─────────────────────────
cd "$PUB_REPO"
if [ "$CURRENT_BRANCH" != "main" ]; then
  echo -e "${BLUE}Switching public repo to branch: $CURRENT_BRANCH${NC}"
  git fetch origin 2>/dev/null || true
  git checkout "$CURRENT_BRANCH" 2>/dev/null || git checkout -b "$CURRENT_BRANCH"
  echo ""
fi
cd "$DEV_REPO"

# ── Step 1: Sync ──────────────────────────────────────────────────────────────
echo -e "${BLUE}Step 1: Running sync script...${NC}"
echo ""
bash "$DEV_REPO/sync-to-public.sh"

echo ""
echo -e "${BLUE}Step 2: Checking for changes in public repo...${NC}"
cd "$PUB_REPO"

# Check if there are changes
if git diff --quiet && git diff --cached --quiet && [ -z "$(git ls-files --others --exclude-standard)" ]; then
  echo -e "${YELLOW}No changes detected. Public repo is already up to date.${NC}"
  exit 0
fi

echo -e "${GREEN}Changes detected!${NC}"
echo ""
echo -e "${YELLOW}Changed files:${NC}"
git status --short
echo ""

# ── Step 3: Verify no sensitive files leaked ─────────────────────────────────
echo -e "${BLUE}Step 3: Verifying no sensitive files leaked...${NC}"
SENSITIVE_PATTERNS=(
  "_dev/"
  ".cursorrules"
  "HANDOFF.md"
  "coverage.out"
  "ayb-linux"
  "ayb-darwin"
  "ayb-windows"
  ".secret"
  "^\.env"
  "agent_logs/"
  "staging.allyourbase.io"
  "stuartcrobinson"
)

LEAK_FOUND=false
for pattern in "${SENSITIVE_PATTERNS[@]}"; do
  if git ls-files | grep -q "$pattern"; then
    echo -e "${RED}ERROR: Sensitive pattern found in tracked files: $pattern${NC}"
    git ls-files | grep "$pattern"
    LEAK_FOUND=true
  fi
done

if [ "$LEAK_FOUND" = "true" ]; then
  echo -e "${RED}Sync verification failed! Fix sync-to-public.sh excludes.${NC}"
  exit 1
fi
echo -e "${GREEN}No sensitive files detected${NC}"
echo ""

# ── Step 4: Commit & Push ─────────────────────────────────────────────────────
echo -e "${BLUE}Step 4: Committing changes...${NC}"
cd "$PUB_REPO"
git add -A
git commit -m "$COMMIT_MESSAGE"

echo ""
echo -e "${BLUE}Step 5: Pushing to GitHub...${NC}"
git push origin "$CURRENT_BRANCH"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Push complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# ── Step 6: Trigger release ──────────────────────────────────────────────────
echo -e "${BLUE}Step 6: Triggering release build...${NC}"

# Get current version from git tags (not releases, which may not exist yet)
CURRENT_VERSION=$(git tag --list 'v*' --sort=-v:refname | head -1 | sed 's/^v//')
CURRENT_VERSION="${CURRENT_VERSION:-0.0.0}"
echo -e "Current version: ${GREEN}v${CURRENT_VERSION}${NC}"

# Parse semver and bump patch (strip any existing pre-release suffix like -beta)
MAJOR=$(echo "$CURRENT_VERSION" | cut -d. -f1)
MINOR=$(echo "$CURRENT_VERSION" | cut -d. -f2)
PATCH=$(echo "$CURRENT_VERSION" | cut -d. -f3 | sed 's/-.*//')
NEW_VERSION="${MAJOR}.${MINOR}.$((PATCH + 1))-beta"

echo -e "New version:     ${GREEN}v${NEW_VERSION}${NC}"
echo ""

git tag "v${NEW_VERSION}"
git push origin "v${NEW_VERSION}"
echo -e "${GREEN}Tag v${NEW_VERSION} pushed — CI + Docker workflows will trigger automatically${NC}"
echo ""

# ── Monitor ──────────────────────────────────────────────────────────────────
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}All done! Monitor progress:${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "  gh run watch --repo $GH_REPO                          # live CI/release updates"
echo "  gh run list --repo $GH_REPO --limit 5                 # recent workflow runs"
echo "  gh release view v${NEW_VERSION} --repo $GH_REPO       # check release when ready"
echo ""
echo "  Repo:    https://github.com/$GH_REPO"
echo "  Actions: https://github.com/$GH_REPO/actions"
echo ""
