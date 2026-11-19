.PHONY: all build test check clean frontend-build generate-ts docs-generate docs-screenshots docs-serve

all: build

build:
	go build -v -o rclone-bisync ./cmd/rclone-bisync

test:
	go test -count=1 ./...

check:
	./scripts/check.sh all

frontend-build:
	cd dashboard/frontend && npm run build

generate-ts:
	go run ./cmd/gendocs --ts-out dashboard/frontend/src/types/models.gen.ts

docs-generate: generate-ts
	go run ./cmd/gendocs --out site/technical-reference.md

docs-screenshots: frontend-build
	CAPTURE_SCREENSHOTS=1 go test -count=1 -v ./tests/e2e -run TestCaptureSandboxScreenshots

docs-serve:
	cd site && bundle exec jekyll serve

clean:
	rm -f rclone-bisync
