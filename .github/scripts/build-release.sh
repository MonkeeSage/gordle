#!/usr/bin/env bash
# Builds release archives of gordle (not gordle-solver) for every supported
# platform, plus a SHA256SUMS file.
#
# Usage: .github/scripts/build-release.sh VERSION OUTDIR   (from the repo root)
set -euo pipefail

version=$1
mkdir -p "$2"
out=$(cd "$2" && pwd)

targets=(linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64)

for target in "${targets[@]}"; do
	os=${target%/*}
	arch=${target#*/}
	name=gordle_${version}_${os}_${arch}
	exe=gordle
	[[ $os == windows ]] && exe=gordle.exe

	work=$(mktemp -d)
	mkdir "$work/$name"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath \
		-ldflags "-s -w -X main.version=$version" -o "$work/$name/$exe" .
	# The binary embeds the SCOWL word lists, whose license requires its
	# notice to accompany every copy.
	cp README.md internal/words/SCOWL-COPYRIGHT "$work/$name/"

	if [[ $os == windows ]]; then
		(cd "$work" && zip -qr "$out/$name.zip" "$name")
	else
		tar -C "$work" -czf "$out/$name.tar.gz" "$name"
	fi
	rm -rf "$work"
	echo "built $name"
done

(cd "$out" && sha256sum -- *.tar.gz *.zip >SHA256SUMS)
