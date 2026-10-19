#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")"
target_os="${GOOS:-linux}"
target_arch="${GOARCH:-amd64}"
destination="../bin/$target_os-$target_arch"
mkdir -p "$destination"
CGO_ENABLED=0 GOOS="$target_os" GOARCH="$target_arch" go build -trimpath -buildvcs=false -ldflags='-s -w' -o "$destination/wcs-repo" .
cd "$destination"
sha256sum wcs-repo > SHA256SUMS
