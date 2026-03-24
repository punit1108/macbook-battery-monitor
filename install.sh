#!/usr/bin/env bash
# mac-battery installer
# Usage: curl -fsSL https://raw.githubusercontent.com/ppunit/mac_battery/main/install.sh | bash
set -euo pipefail

REPO_URL="https://github.com/ppunit/mac_battery.git"
BIN_NAME="mac-battery"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# ── Output helpers ────────────────────────────────────────────────────────────
if [ -t 1 ]; then
  CY='\033[0;36m' GR='\033[0;32m' YL='\033[0;33m' RD='\033[0;31m' BD='\033[1m' RS='\033[0m'
else
  CY='' GR='' YL='' RD='' BD='' RS=''
fi

step() { printf "${CY}▶${RS}  %s\n"   "$*"; }
ok()   { printf "${GR}✓${RS}  %s\n"   "$*"; }
warn() { printf "${YL}!${RS}  %s\n"   "$*"; }
die()  { printf "${RD}✗${RS}  %s\n"   "$*" >&2; exit 1; }
br()   { printf "\n"; }

# ── Platform check ────────────────────────────────────────────────────────────
[ "$(uname)" = "Darwin" ] || die "mac-battery only supports macOS."

ARCH="$(uname -m)"
case "$ARCH" in
  arm64|x86_64) ;;
  *) die "Unsupported architecture: $ARCH" ;;
esac

ok "macOS $(sw_vers -productVersion) on $ARCH"

# ── Dependency: git ───────────────────────────────────────────────────────────
if ! command -v git >/dev/null 2>&1; then
  # Xcode CLT prompt will appear automatically on first git call; abort cleanly.
  die "git is required. Run:  xcode-select --install"
fi

# ── Dependency: Go ────────────────────────────────────────────────────────────
ensure_go() {
  if command -v go >/dev/null 2>&1; then
    GO_VER="$(go version | awk '{print $3}')"
    ok "Go $GO_VER"
    return
  fi

  warn "Go is not installed."

  if command -v brew >/dev/null 2>&1; then
    step "Installing Go via Homebrew..."
    brew install go
    ok "Go $(go version | awk '{print $3}')"
  else
    br
    printf "Go is required to build mac-battery.\n"
    printf "Install options:\n"
    printf "  Homebrew (recommended):  https://brew.sh  then  brew install go\n"
    printf "  Direct download:         https://go.dev/dl\n"
    br
    die "Please install Go and re-run this script."
  fi
}

ensure_go

# ── Build from source ─────────────────────────────────────────────────────────
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

step "Cloning repository..."
git clone --quiet --depth 1 "$REPO_URL" "$TMP/src" \
  || die "Failed to clone $REPO_URL — check your internet connection."
ok "Source ready"

step "Building mac-battery (this may take a moment on first run)..."
(
  cd "$TMP/src"
  GOFLAGS="-mod=mod" go build -ldflags="-s -w" -o "$TMP/$BIN_NAME" . 2>&1
) || die "Build failed."
ok "Build complete"

# ── Install binary ────────────────────────────────────────────────────────────
mkdir -p "$INSTALL_DIR"

EXISTING=""
if command -v "$BIN_NAME" >/dev/null 2>&1; then
  EXISTING="$(command -v "$BIN_NAME")"
fi

install -m 755 "$TMP/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
ok "Installed → $INSTALL_DIR/$BIN_NAME"

if [ -n "$EXISTING" ] && [ "$EXISTING" != "$INSTALL_DIR/$BIN_NAME" ]; then
  warn "Another copy exists at $EXISTING — you may want to remove it."
fi

# ── PATH setup ────────────────────────────────────────────────────────────────
path_line='export PATH="$HOME/.local/bin:$PATH"'

add_to_shell_rc() {
  local rc="$1"
  if [ -f "$rc" ] && grep -q '\.local/bin' "$rc" 2>/dev/null; then
    return  # already present
  fi
  printf '\n# Added by mac-battery installer\n%s\n' "$path_line" >> "$rc"
  ok "Added ~/.local/bin to PATH in $rc"
}

if ! printf "%s" "$PATH" | grep -q "$HOME/.local/bin"; then
  warn "~/.local/bin is not in your PATH."
  SHELL_NAME="$(basename "${SHELL:-/bin/zsh}")"
  case "$SHELL_NAME" in
    zsh)  add_to_shell_rc "$HOME/.zshrc"       ;;
    bash) add_to_shell_rc "$HOME/.bash_profile" ;;
    *)    add_to_shell_rc "$HOME/.profile"      ;;
  esac
  warn "Run this to use mac-battery in your current session:"
  printf "      export PATH=\"\$HOME/.local/bin:\$PATH\"\n"
fi

# ── Daemon prompt ─────────────────────────────────────────────────────────────
br
printf "${BD}Set up background daemon?${RS}\n"
printf "The daemon runs silently in the background, collecting battery analytics\n"
printf "every 60 seconds. It auto-starts on login and uses <0.1%% CPU on average.\n"
printf "This is required for the App Drain history tab to work when the TUI is closed.\n"
br

# Read from /dev/tty so this works even when piped through curl | bash
ANSWER="y"
if [ -c /dev/tty ]; then
  printf "Install daemon? [Y/n] "
  read -r ANSWER </dev/tty || ANSWER="y"
else
  printf "Install daemon? [Y/n] y (non-interactive)\n"
fi

case "${ANSWER:-y}" in
  [Yy]*)
    step "Installing LaunchAgent..."
    "$INSTALL_DIR/$BIN_NAME" install \
      || die "Daemon install failed. Try running:  mac-battery install"
    ;;
  *)
    warn "Skipped. Run  mac-battery install  whenever you're ready."
    ;;
esac

# ── Done ──────────────────────────────────────────────────────────────────────
br
printf "${GR}${BD}All done!${RS} mac-battery is installed.\n"
br
printf "  ${CY}mac-battery${RS}             Open the TUI\n"
printf "  ${CY}mac-battery install${RS}     Start background data collection\n"
printf "  ${CY}mac-battery uninstall${RS}   Remove daemon and binary\n"
br
