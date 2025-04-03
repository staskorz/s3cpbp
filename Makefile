.PHONY: build clean windows linux darwin install test coverage

build: test
	@echo "Building for all platforms..."
	@chmod +x scripts/build.sh
	@./scripts/build.sh

test:
	@echo "Running tests..."
	@go test ./... -v

coverage:
	@echo "Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -func=coverage.out
	@echo "Checking coverage threshold..."
	@go tool cover -func=coverage.out | grep total | awk '{print $$3}' | tr -d '%' | awk '{if ($$1 < 70) {print "Code coverage is below 70%"; exit 1}}'
	@echo "Generating HTML coverage report..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

clean:
	@echo "Cleaning previous builds..."
	@chmod +x scripts/build.sh
	@./scripts/build.sh clean
	@rm -f coverage.out coverage.html

windows: test
	@echo "Building for Windows..."
	@scripts/build.bat

linux: test
	@echo "Building for Linux..."
	@chmod +x scripts/build.sh
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=0.1.0" -o "./bin/s3cpbp_linux_amd64" ./cmd/s3cpbp

darwin: test
	@echo "Building for macOS..."
	@chmod +x scripts/build.sh
	@GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=0.1.0" -o "./bin/s3cpbp_darwin_amd64" ./cmd/s3cpbp
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=0.1.0" -o "./bin/s3cpbp_darwin_arm64" ./cmd/s3cpbp

install: test
	@echo "Installing to GOPATH/bin..."
	@go install -ldflags="-s -w -X main.version=0.1.0" ./cmd/s3cpbp 