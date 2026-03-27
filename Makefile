.PHONY: setup lint lint-fix test build generate-themes

setup:
	git config core.hooksPath .githooks

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

test:
	go test ./...

build: setup
	go build -o lfk .

GHOSTTY_THEMES_URL := https://deps.files.ghostty.org/ghostty-themes-release-20260216-151611-fc73ce3.tgz
GHOSTTY_THEMES_DIR := themes/ghostty

generate-themes: ## Download ghostty themes and regenerate colorschemes_gen.go
	@echo "Downloading ghostty themes..."
	@mkdir -p themes
	@curl -sL $(GHOSTTY_THEMES_URL) | tar xz -C themes/
	@echo "Generating colorschemes..."
	go run ./cmd/themegen --input-dir=$(GHOSTTY_THEMES_DIR) --output=internal/ui/colorschemes_gen.go
	@echo "Done. Run 'go test ./internal/ui/' to verify."
