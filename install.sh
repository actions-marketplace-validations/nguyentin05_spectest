#!/usr/bin/env bash
# SpecTest-AI Installation Script
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/nguyentin05/spectest/main/install.sh | sh
#
# Custom install location:
#   curl -fsSL https://raw.githubusercontent.com/nguyentin05/spectest/main/install.sh | INSTALL_DIR=~/.local/bin sh

set -euo pipefail

REPO="nguyentin05/spectest"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${SPECTEST_VERSION:-latest}"

# 1. Detect OS and Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    echo "Error: Unsupported operating system: $OS. SpecTest supports Linux and macOS via this installer." >&2
    exit 1
    ;;
esac

case "$ARCH" in
  x86_64|amd64)   ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# 2. Resolve version tag
if [ "$VERSION" = "latest" ]; then
  echo "Resolving latest release for ${REPO}..."
  LATEST_URL="$(curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest")"
  VERSION="${LATEST_URL##*/}"
  if [ -z "$VERSION" ] || [ "$VERSION" = "latest" ]; then
    echo "Error: Unable to resolve latest release tag from GitHub." >&2
    exit 1
  fi
fi

# Ensure leading 'v' in version tag
[[ "$VERSION" == v* ]] || VERSION="v${VERSION}"
CLEAN_VERSION="${VERSION#v}"

ARCHIVE="spectest_${CLEAN_VERSION}_${OS}_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

echo "Installing SpecTest ${VERSION} (${OS}/${ARCH})..."

# 3. Create temporary directory
TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

# 4. Download archive and checksums
echo "Downloading ${DOWNLOAD_URL}..."
curl -fsSL --retry 3 -o "${TMP_DIR}/${ARCHIVE}" "$DOWNLOAD_URL"
curl -fsSL --retry 3 -o "${TMP_DIR}/checksums.txt" "$CHECKSUM_URL" || true

# 5. Verify checksum if available
if [ -f "${TMP_DIR}/checksums.txt" ]; then
  EXPECTED="$(grep -E "[[:space:]]${ARCHIVE}$" "${TMP_DIR}/checksums.txt" | awk '{print $1}')"
  if [ -n "$EXPECTED" ]; then
    if command -v sha256sum >/dev/null 2>&1; then
      ACTUAL="$(sha256sum "${TMP_DIR}/${ARCHIVE}" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
      ACTUAL="$(shasum -a 256 "${TMP_DIR}/${ARCHIVE}" | awk '{print $1}')"
    else
      ACTUAL=""
    fi

    if [ -n "$ACTUAL" ]; then
      if [ "$EXPECTED" != "$ACTUAL" ]; then
        echo "Error: Checksum verification failed!" >&2
        echo "Expected: $EXPECTED" >&2
        echo "Got:      $ACTUAL" >&2
        exit 1
      fi
      echo "Checksum verified (SHA-256: ${ACTUAL:0:16}...)"
    fi
  fi
fi

# 6. Extract binary
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR" spectest

# 7. Install binary
mkdir -p "$INSTALL_DIR"
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP_DIR}/spectest" "${INSTALL_DIR}/spectest"
  chmod +x "${INSTALL_DIR}/spectest"
else
  echo "Elevated permissions required to install to ${INSTALL_DIR}..."
  sudo mv "${TMP_DIR}/spectest" "${INSTALL_DIR}/spectest"
  sudo chmod +x "${INSTALL_DIR}/spectest"
fi

echo "SpecTest ${VERSION} successfully installed to ${INSTALL_DIR}/spectest"

# 8. Check PATH
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "Note: '${INSTALL_DIR}' is not currently in your PATH."
    echo "Add it by running:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    echo ""
    ;;
esac

# 9. Verify
if command -v spectest >/dev/null 2>&1; then
  spectest --version
fi
