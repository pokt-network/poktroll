#!/bin/bash
set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RESET='\033[0m'

echo -e "${CYAN}════════════════════════════════════════════════════════${RESET}"
echo -e "${CYAN}     Docker CI Readiness Check for Poktroll${RESET}"
echo -e "${CYAN}════════════════════════════════════════════════════════${RESET}"
echo ""

ERRORS=0
WARNINGS=0

# Check 1: Linux binaries exist
echo -e "${CYAN}1. Checking for Linux binaries...${RESET}"
if ls release_binaries/pocket_linux_* >/dev/null 2>&1; then
    echo -e "${GREEN}   ✅ Found non-CGO Linux binaries:${RESET}"
    ls -lh release_binaries/pocket_linux_* | awk '{print "      " $9 " (" $5 ")"}'
else
    echo -e "${RED}   ❌ Missing Linux binaries${RESET}"
    echo "      Run: make ignite_release_cgo_disabled"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# Check 2: Test actual Dockerfile with real binaries
echo -e "${CYAN}2. Testing Dockerfile.release with actual binaries...${RESET}"
if docker build -f Dockerfile.release -t test:validation . >/dev/null 2>&1; then
    echo -e "${GREEN}   ✅ Dockerfile build successful${RESET}"

    # Test if image runs
    if docker run --rm test:validation version >/dev/null 2>&1; then
        echo -e "${GREEN}   ✅ Docker image executes correctly${RESET}"
    else
        echo -e "${RED}   ❌ Docker image fails to run${RESET}"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo -e "${RED}   ❌ Dockerfile build failed${RESET}"
    echo "      Debug with: docker build -f Dockerfile.release . --progress=plain"
    ERRORS=$((ERRORS + 1))
fi
echo ""

# Check 3: Check ignite.mk for CGO naming fix
echo -e "${CYAN}3. Verifying CGO binary naming fix...${RESET}"
if grep -q 'pocket_cgo_[[:space:]]' makefiles/ignite.mk 2>/dev/null; then
    echo -e "${RED}   ❌ Found trailing underscore in CGO prefix (will create double underscore)${RESET}"
    echo "      File: makefiles/ignite.mk"
    grep -n 'pocket_cgo_' makefiles/ignite.mk | head -2
    ERRORS=$((ERRORS + 1))
elif grep -q 'pocket_cgo[[:space:]]' makefiles/ignite.mk 2>/dev/null; then
    echo -e "${GREEN}   ✅ CGO prefix correctly set (no trailing underscore)${RESET}"
else
    echo -e "${YELLOW}   ⚠️  Could not verify CGO prefix${RESET}"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# Check 4: Verify Dockerfile handles both naming patterns
echo -e "${CYAN}4. Checking Dockerfile handles multiple naming patterns...${RESET}"
if grep -q 'pocket_cgo__linux' Dockerfile.release && grep -q 'pocket_cgo_linux' Dockerfile.release; then
    echo -e "${GREEN}   ✅ Dockerfile handles both naming patterns${RESET}"
    echo "      - Old pattern: pocket_cgo__linux_\${TARGETARCH}"
    echo "      - New pattern: pocket_cgo_linux_\${TARGETARCH}"
else
    echo -e "${YELLOW}   ⚠️  Dockerfile might not handle all naming patterns${RESET}"
    echo "      Check binary selection logic in Dockerfile.release"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# Check 5: Disk cleanup in workflow
echo -e "${CYAN}5. Checking for disk cleanup in CI workflow...${RESET}"
if grep -q "Free disk space" .github/workflows/release-artifacts.yml 2>/dev/null; then
    echo -e "${GREEN}   ✅ Disk cleanup step found in workflow${RESET}"
else
    echo -e "${YELLOW}   ⚠️  No disk cleanup step in workflow${RESET}"
    echo "      Consider adding cleanup to prevent 'no space left' errors"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# Check 6: Multi-stage build optimization
echo -e "${CYAN}6. Verifying multi-stage Docker build...${RESET}"
if grep -q "FROM.*AS binary-selector" Dockerfile.release; then
    echo -e "${GREEN}   ✅ Multi-stage build implemented (reduces disk usage)${RESET}"
else
    echo -e "${YELLOW}   ⚠️  Not using multi-stage build${RESET}"
    echo "      Multi-stage builds help reduce disk usage in CI"
    WARNINGS=$((WARNINGS + 1))
fi
echo ""

# Summary
echo -e "${CYAN}════════════════════════════════════════════════════════${RESET}"
echo -e "${CYAN}                    SUMMARY${RESET}"
echo -e "${CYAN}════════════════════════════════════════════════════════${RESET}"

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}✨ All checks passed! Safe to push to CI.${RESET}"
elif [ $ERRORS -eq 0 ]; then
    echo -e "${YELLOW}⚠️  No errors, but $WARNINGS warning(s) found.${RESET}"
    echo -e "${YELLOW}   You can proceed, but review warnings above.${RESET}"
else
    echo -e "${RED}❌ Found $ERRORS error(s) and $WARNINGS warning(s).${RESET}"
    echo -e "${RED}   Fix errors before pushing to CI.${RESET}"
fi

echo ""
echo -e "${CYAN}Quick test command before pushing:${RESET}"
echo "  docker buildx build --platform linux/amd64 -f Dockerfile.release ."
echo ""

# Cleanup
docker rmi test:validation >/dev/null 2>&1 || true

exit $ERRORS