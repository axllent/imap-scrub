#!/usr/bin/env bash

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
project_dir=$(CDPATH= cd -- "$script_dir/.." && pwd)

mkdir -p "$project_dir/dist"

docker run --rm \
    --user "$(id -u):$(id -g)" \
    --env CGO_ENABLED=0 \
    --env GOOS=linux \
    --env GOARCH=amd64 \
    --env GOCACHE=/tmp/go-build \
    --env GOMODCACHE=/tmp/go-mod \
    --volume "$project_dir:/workspace" \
    --workdir /workspace \
    golang:1.20 \
    go build -ldflags="-s -w" -o "dist/imap-scrub-linux-amd64" .
