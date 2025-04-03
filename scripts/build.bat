@echo off
setlocal

set VERSION=0.1.0
set BINARY_NAME=s3cpbp
set OUTPUT_DIR=.\bin

:: Create output directory if it doesn't exist
if not exist %OUTPUT_DIR% mkdir %OUTPUT_DIR%

:: Run tests
:run_tests
echo Running tests...
go test .\... -v
if %ERRORLEVEL% neq 0 (
    echo Tests failed
    exit /b 1
)

:: Run tests with coverage
:check_coverage
if "%1"=="coverage" (
    echo Running tests with coverage...
    go test .\... -coverprofile=coverage.out
    if %ERRORLEVEL% neq 0 exit /b 1
    
    go tool cover -func=coverage.out
    
    :: Generate HTML coverage report
    go tool cover -html=coverage.out -o coverage.html
    echo Coverage report generated at coverage.html
    exit /b 0
)

:: Clean if requested
if "%1"=="clean" (
    echo Cleaning up previous builds...
    if exist %OUTPUT_DIR% rmdir /s /q %OUTPUT_DIR%
    mkdir %OUTPUT_DIR%
    if exist coverage.out del coverage.out
    if exist coverage.html del coverage.html
    goto :eof
)

:: Run only tests if requested
if "%1"=="test" (
    goto run_tests
)

echo Building s3cpbp v%VERSION%...

:: Build for Windows (current platform)
echo Building for Windows/amd64...
set GOOS=windows
set GOARCH=amd64
go build -ldflags="-s -w -X main.version=%VERSION%" -o "%OUTPUT_DIR%\%BINARY_NAME%.exe" .\cmd\s3cpbp

echo Build completed!
echo Binary is available at %OUTPUT_DIR%\%BINARY_NAME%.exe

endlocal 