# Makefile for building the interceptor

# Target directory for the build output
BUILD_DIR := build

.PHONY: build-interceptor

build-interceptor:
	@echo "Building the interceptor..."
	@go build -o $(BUILD_DIR)/interceptor ./cmd/interceptor/main.go
	@echo "Build complete!"
