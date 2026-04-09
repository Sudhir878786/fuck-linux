#!/usr/bin/env bash
# fucklinux — one-line installer
# Usage: curl -fsSL https://raw.githubusercontent.com/Sudhir878786/fuck-linux/Sudhir/install.sh | bash

set -e

REPO="Sudhir878786/fuck-linux"
BINARY="fuck-linux"
INSTALL_DIR="/usr/local/bin"
SOUND_DIR="/usr/share/fucklinux/sounds"

echo ""
echo "╔══════════════════════════════════════╗"
echo "║   Installing fucklinux...  🔊        ║"
echo "╚══════════════════════════════════════╝"
echo ""

# Detect OS and architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  armv7l)  ARCH="armv7" ;;
  *)
    echo "❌ Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

if [ "$OS" != "linux" ]; then
  echo "❌ fucklinux only supports Linux."
  exit 1
fi

# Get latest release tag from GitHub API
echo "→ Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
  | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "❌ Could not fetch latest release. Check your internet connection."
  exit 1
fi

echo "→ Latest version: $LATEST"

TARBALL="fucklinux_${LATEST#v}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST}/${TARBALL}"

# Download and extract
TMP=$(mktemp -d)
trap "rm -rf $TMP" EXIT

echo "→ Downloading $TARBALL..."
curl -fsSL "$URL" -o "$TMP/fucklinux.tar.gz"
tar -xzf "$TMP/fucklinux.tar.gz" -C "$TMP"

# Install binary
echo "→ Installing binary to $INSTALL_DIR (may require sudo)..."
sudo install -Dm755 "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"

# Install sounds
echo "→ Installing sounds to $SOUND_DIR..."
sudo mkdir -p "$SOUND_DIR"
if [ -d "$TMP/sounds" ]; then
  sudo cp -r "$TMP/sounds/." "$SOUND_DIR/"
fi

echo ""
echo "✅ fucklinux installed successfully!"
echo ""
echo "Usage:"
echo "  $BINARY --source mic --sound-dir $SOUND_DIR --threshold 0.4"
echo ""
echo "Linux moans every time you hit it. 🎉"
echo ""
