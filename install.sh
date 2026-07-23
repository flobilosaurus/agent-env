#!/bin/sh
set -eu

repo="${AGENTENV_REPO:-flobilosaurus/agent-env}"
version="${AGENTENV_VERSION:-latest}"
install_dir="${AGENTENV_INSTALL_DIR:-$HOME/.local/bin}"
bin_name="agentenv"

die() {
  echo "agentenv install: $*" >&2
  exit 1
}

need() {
  command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

need curl
need tar
need grep
need awk
need install

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  darwin|linux) ;;
  *) die "unsupported OS: $os" ;;
esac

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) die "unsupported architecture: $arch" ;;
esac

asset="agentenv-${os}-${arch}.tar.gz"
if [ "$version" = "latest" ]; then
  url="https://github.com/${repo}/releases/latest/download/${asset}"
  checksums_url="https://github.com/${repo}/releases/latest/download/checksums.txt"
else
  url="https://github.com/${repo}/releases/download/${version}/${asset}"
  checksums_url="https://github.com/${repo}/releases/download/${version}/checksums.txt"
fi

tmp="$(mktemp -d 2>/dev/null || mktemp -d -t agentenv)"
trap 'rm -rf "$tmp"' EXIT INT TERM

archive="$tmp/$asset"
checksums="$tmp/checksums.txt"

echo "Downloading $url"
curl -fsSL "$url" -o "$archive"
curl -fsSL "$checksums_url" -o "$checksums"

expected="$(grep "[[:space:]]$asset$" "$checksums" | awk '{print $1}')"
[ -n "$expected" ] || die "checksum not found for $asset"

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$archive" | awk '{print $1}')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
else
  die "missing sha256sum or shasum"
fi

[ "$actual" = "$expected" ] || die "checksum mismatch for $asset"

mkdir -p "$install_dir"
tar -xzf "$archive" -C "$tmp"
install -m 755 "$tmp/$bin_name" "$install_dir/$bin_name"

echo "Installed $bin_name to $install_dir/$bin_name"
echo "Make sure $install_dir is on your PATH."
