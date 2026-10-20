#!/bin/bash
set -ex

# The Go binary embeds ./dist (go:embed all:dist). The committed placeholder
# keeps fresh clones buildable; this guard also covers local builds after a
# `rm -rf dist`. A real `npm run build` replaces the placeholder.
mkdir -p dist
[ -f dist/index.html ] || printf '<!doctype html><title>metahub-admin-ui</title>placeholder\n' > dist/index.html

# UI tests (skipped when node_modules is absent — e.g. Go-only CI stages).
if [ -d node_modules ]; then
  npm run test:coverage
fi

echo "Running Go tests with coverage..."
# `./...` would sweep Go code shipped inside node_modules (e.g. flatted);
# restrict to our own packages.
go test -short -timeout 120s $(go list ./... | grep -v /node_modules/) -coverprofile coverage.txt
gocover-cobertura < coverage.txt > coverage.xml
gocoverstats -v -f coverage.txt --percent
