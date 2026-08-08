#!/usr/bin/env sh
set -eu
GOCACHE=${GOCACHE:-/tmp/gai-go-cache}
export GOCACHE
gofmt -w cmd internal
go vet ./...
go test ./...
cd web
npm run lint
npm test
npm run build
cd ..
./scripts/verify_iteration_archive.sh
