.PHONY: build build-extension
build:
	go build ./cmd/codeowners

# Build the GitHub CLI extension binary (gh-codeowners)
build-extension:
	go build -o gh-codeowners ./cmd/gh-codeowners

