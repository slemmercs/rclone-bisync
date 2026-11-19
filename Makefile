.PHONY: all build test check clean docs-generate docs-screenshots docs-serve

all: build

build:
	go build -v -o rclone-bisync ./cmd/rclone-bisync

test:
	go test -count=1 ./...

check:
	./scripts/check.sh all

docs-generate:
	go run ./cmd/gendocs --out site/technical-reference.md

docs-screenshots:
	go test -count=1 -v ./tests/e2e -run TestCaptureSandboxScreenshots

docs-serve:
	cd site && bundle exec jekyll serve

clean:
	rm -f rclone-bisync
