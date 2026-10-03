#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version=$(tr -d '\n' < VERSION)
if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then echo 'VERSION must be MAJOR.MINOR.PATCH with an optional prerelease suffix' >&2; exit 1; fi
if [[ -n "${CI_COMMIT_TAG:-}" && "$CI_COMMIT_TAG" != "v$version" ]]; then echo 'Release tag must match VERSION' >&2; exit 1; fi
repository=${SLOPMETER_RELEASE_REPOSITORY:-https://github.com/Netskill89/slopmeter}
mkdir -p build
CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=$version -X main.releaseRepository=$repository" -o build/slopmeter-capture ./cmd/slopmeter-capture
cmake -S ui -B build/ui -DCMAKE_BUILD_TYPE=Release -DSLOPMETER_VERSION="$version" -DSLOPMETER_RELEASE_REPOSITORY="$repository"
cmake --build build/ui --parallel "${BUILD_JOBS:-2}"
