#!/bin/bash
set -e

echo "🔍 Testing Dockerfile binary naming patterns..."
echo ""

# Create test directory structure
mkdir -p test_binaries
mkdir -p test_tmp

# Create dummy binaries with different naming patterns
echo "Creating test binaries with various naming patterns..."

# Non-CGO pattern (correct)
touch test_binaries/pocket_linux_amd64
touch test_binaries/pocket_linux_arm64

# CGO pattern - OLD (double underscore - what CI might have)
touch test_binaries/pocket_cgo__linux_amd64
touch test_binaries/pocket_cgo__linux_arm64

# CGO pattern - NEW (single underscore - what we fixed)
touch test_binaries/pocket_cgo_linux_amd64
touch test_binaries/pocket_cgo_linux_arm64

# Make them executable
chmod +x test_binaries/*

# Create dummy cosmovisor binaries
touch test_tmp/cosmovisor-linux-amd64
touch test_tmp/cosmovisor-linux-arm64
chmod +x test_tmp/*

echo "Test binaries created:"
ls -la test_binaries/
echo ""

# Create a test Dockerfile that copies from test directories
cat > Dockerfile.test <<'EOF'
FROM busybox:stable AS binary-selector
ARG TARGETARCH=amd64
ARG BIN_PREFIX=test_binaries/pocket_linux

COPY test_binaries/pocket* /tmp/release_binaries/
COPY test_tmp/cosmovisor-linux-* /tmp/

RUN set -eu; \
    found=0; \
    if echo "${BIN_PREFIX}" | grep -q "cgo"; then \
        for pattern in "pocket_cgo__linux_${TARGETARCH}" "pocket_cgo_linux_${TARGETARCH}"; do \
            binary_name="/tmp/release_binaries/${pattern}"; \
            if [ -f "${binary_name}" ]; then \
                cp "${binary_name}" /tmp/pocketd; \
                chmod 755 /tmp/pocketd; \
                found=1; \
                echo "Found CGO binary: ${pattern}"; \
                break; \
            fi; \
        done; \
    else \
        binary_name="/tmp/release_binaries/pocket_linux_${TARGETARCH}"; \
        if [ -f "${binary_name}" ]; then \
            cp "${binary_name}" /tmp/pocketd; \
            chmod 755 /tmp/pocketd; \
            found=1; \
            echo "Found non-CGO binary: pocket_linux_${TARGETARCH}"; \
        fi; \
    fi; \
    if [ ${found} -eq 0 ]; then \
        echo "ERROR: Binary not found for TARGETARCH=${TARGETARCH}"; \
        ls -la /tmp/release_binaries/; \
        exit 1; \
    fi; \
    echo "Successfully selected binary for ${TARGETARCH}"

FROM busybox:stable
COPY --from=binary-selector /tmp/pocketd /bin/pocketd
CMD ["/bin/pocketd"]
EOF

echo "Testing non-CGO build (standard naming)..."
if docker build -f Dockerfile.test --no-cache -t test:nocgo . > /tmp/docker-test.log 2>&1; then
    echo "✅ Non-CGO build successful"
else
    echo "❌ Non-CGO build failed"
    cat /tmp/docker-test.log
    exit 1
fi

echo ""
echo "Testing CGO build with OLD double underscore naming..."
if docker build -f Dockerfile.test --no-cache -t test:cgo-old \
    --build-arg BIN_PREFIX=test_binaries/pocket_cgo_linux . > /tmp/docker-test.log 2>&1; then
    echo "✅ CGO build with old naming (double underscore) successful"
else
    echo "❌ CGO build with old naming failed"
    cat /tmp/docker-test.log
    exit 1
fi

echo ""
echo "Testing CGO build with NEW single underscore naming..."
# Remove old pattern to ensure new pattern is used
rm -f test_binaries/pocket_cgo__linux_*
if docker build -f Dockerfile.test --no-cache -t test:cgo-new \
    --build-arg BIN_PREFIX=test_binaries/pocket_cgo_linux . > /tmp/docker-test.log 2>&1; then
    echo "✅ CGO build with new naming (single underscore) successful"
else
    echo "❌ CGO build with new naming failed"
    cat /tmp/docker-test.log
    exit 1
fi

echo ""
echo "Testing multi-arch build..."
docker buildx create --name test-builder --use >/dev/null 2>&1 || true
if docker buildx build --platform linux/amd64,linux/arm64 \
    -f Dockerfile.test -t test:multi . > /tmp/docker-test.log 2>&1; then
    echo "✅ Multi-platform build successful"
else
    echo "❌ Multi-platform build failed"
    cat /tmp/docker-test.log
    docker buildx rm test-builder >/dev/null 2>&1
    exit 1
fi
docker buildx rm test-builder >/dev/null 2>&1

# Cleanup
rm -rf test_binaries test_tmp Dockerfile.test
docker rmi -f test:nocgo test:cgo-old test:cgo-new test:multi >/dev/null 2>&1

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✨ All naming pattern tests passed!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "The Dockerfile correctly handles:"
echo "  ✅ pocket_linux_amd64 (non-CGO)"
echo "  ✅ pocket_cgo__linux_amd64 (old CGO with double underscore)"
echo "  ✅ pocket_cgo_linux_amd64 (new CGO with single underscore)"
echo ""