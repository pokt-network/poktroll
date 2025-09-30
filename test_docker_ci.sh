#!/bin/bash
set -e

echo "🔍 Starting Docker CI validation tests..."
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
CYAN='\033[0;36m'
RESET='\033[0m'

# Test 1: Check binaries exist
echo -e "${CYAN}Test 1: Checking for required binaries...${RESET}"
if ls release_binaries/pocket_linux_* >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Linux binaries found${RESET}"
    ls -lh release_binaries/pocket_linux_*
else
    echo -e "${RED}❌ Linux binaries missing${RESET}"
    exit 1
fi
echo ""

# Test 2: Test basic Docker build
echo -e "${CYAN}Test 2: Testing basic Docker build...${RESET}"
if docker build -f Dockerfile.release -t test:basic . --quiet; then
    echo -e "${GREEN}✅ Basic Docker build successful${RESET}"
else
    echo -e "${RED}❌ Basic Docker build failed${RESET}"
    exit 1
fi
echo ""

# Test 3: Test image runs
echo -e "${CYAN}Test 3: Testing Docker image execution...${RESET}"
if docker run --rm test:basic version >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Docker image runs successfully${RESET}"
    docker run --rm test:basic version
else
    echo -e "${RED}❌ Docker image failed to run${RESET}"
    exit 1
fi
echo ""

# Test 4: Test CGO build (if binaries exist)
echo -e "${CYAN}Test 4: Testing CGO Docker build...${RESET}"
if ls release_binaries/pocket_cgo*linux* >/dev/null 2>&1; then
    if docker build -f Dockerfile.release -t test:cgo \
        --build-arg BIN_PREFIX=release_binaries/pocket_cgo_linux . --quiet; then
        echo -e "${GREEN}✅ CGO Docker build successful${RESET}"
    else
        echo -e "${RED}❌ CGO Docker build failed${RESET}"
        exit 1
    fi
else
    echo -e "${CYAN}⏩ Skipping CGO test (no CGO binaries found)${RESET}"
fi
echo ""

# Test 5: Multi-platform simulation
echo -e "${CYAN}Test 5: Testing multi-platform build (CI simulation)...${RESET}"
echo "Creating temporary buildx builder..."
docker buildx create --name ci-test --use >/dev/null 2>&1 || true

if docker buildx build --platform linux/amd64 \
    -f Dockerfile.release -t test:multi . --load --quiet; then
    echo -e "${GREEN}✅ Multi-platform build successful${RESET}"
else
    echo -e "${RED}❌ Multi-platform build failed${RESET}"
    docker buildx rm ci-test >/dev/null 2>&1 || true
    exit 1
fi

# Cleanup
docker buildx rm ci-test >/dev/null 2>&1 || true
echo ""

# Summary
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo -e "${GREEN}✨ All tests passed! Safe to push to CI${RESET}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
echo ""
echo "Cleanup: Run 'docker rmi test:basic test:cgo test:multi' to remove test images"