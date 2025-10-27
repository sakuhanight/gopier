@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

REM Gopier Build Script (Batch Version)
REM Usage: build.bat [action]

set "BINARY_NAME=gopier.exe"
set "BUILD_DIR=build"
set "GO_PATH=C:\Program Files\Go\bin"

REM Add Go to PATH
set "PATH=%GO_PATH%;%PATH%"

REM Get action
set "ACTION=%1"
if "%ACTION%"=="" set "ACTION=build"

REM Get version info
for /f "delims=" %%i in ('git describe --tags --always --dirty 2^>nul') do set "VERSION=%%i"
if "%VERSION%"=="" set "VERSION=dev"

REM Get build time
for /f "tokens=1-3 delims=/ " %%a in ('date /t') do set "BUILD_DATE=%%a-%%b-%%c"
for /f "tokens=1-2 delims=: " %%a in ('time /t') do set "BUILD_TIME=%%a-%%b"
set "BUILD_TIME_FULL=%BUILD_DATE%_%BUILD_TIME%"

echo Gopier Build Script
echo ===================

REM Check Go command
go version >nul 2>&1
if errorlevel 1 (
    echo Error: Go is not installed or not in PATH
    echo Please install Go: https://golang.org/dl/
    exit /b 1
)

goto :%ACTION%

:build
echo Building...
go build -ldflags "-X github.com/sakuhanight/gopier/cmd.Version=%VERSION% -X github.com/sakuhanight/gopier/cmd.BuildTime=%BUILD_TIME_FULL%" -o %BINARY_NAME%
if exist %BINARY_NAME% (
    for %%A in (%BINARY_NAME%) do set "SIZE=%%~zA"
    set /a "SIZE_MB=!SIZE!/1048576"
    echo Build complete: %BINARY_NAME% (!SIZE_MB! MB)
) else (
    echo Error: Build failed
    exit /b 1
)
goto :end

:release
echo Building release version...
go build -ldflags "-s -w -X github.com/sakuhanight/gopier/cmd.Version=%VERSION% -X github.com/sakuhanight/gopier/cmd.BuildTime=%BUILD_TIME_FULL%" -o %BINARY_NAME%
if exist %BINARY_NAME% (
    for %%A in (%BINARY_NAME%) do set "SIZE=%%~zA"
    set /a "SIZE_MB=!SIZE!/1048576"
    echo Release build complete: %BINARY_NAME% (!SIZE_MB! MB)
) else (
    echo Error: Release build failed
    exit /b 1
)
goto :end

:cross-build
echo Building for multiple platforms...
if not exist %BUILD_DIR% mkdir %BUILD_DIR%

echo Windows AMD64...
set "GOOS=windows"
set "GOARCH=amd64"
go build -ldflags "-X github.com/sakuhanight/gopier/cmd.Version=%VERSION% -X github.com/sakuhanight/gopier/cmd.BuildTime=%BUILD_TIME_FULL%" -o %BUILD_DIR%\gopier-windows-amd64.exe
if errorlevel 1 echo   ✗ Windows AMD64 build failed

echo Linux AMD64...
set "GOOS=linux"
set "GOARCH=amd64"
go build -ldflags "-X github.com/sakuhanight/gopier/cmd.Version=%VERSION% -X github.com/sakuhanight/gopier/cmd.BuildTime=%BUILD_TIME_FULL%" -o %BUILD_DIR%\gopier-linux-amd64
if errorlevel 1 echo   ✗ Linux AMD64 build failed

echo macOS AMD64...
set "GOOS=darwin"
set "GOARCH=amd64"
go build -ldflags "-X github.com/sakuhanight/gopier/cmd.Version=%VERSION% -X github.com/sakuhanight/gopier/cmd.BuildTime=%BUILD_TIME_FULL%" -o %BUILD_DIR%\gopier-darwin-amd64
if errorlevel 1 echo   ✗ macOS AMD64 build failed

echo macOS ARM64...
set "GOOS=darwin"
set "GOARCH=arm64"
go build -ldflags "-X github.com/sakuhanight/gopier/cmd.Version=%VERSION% -X github.com/sakuhanight/gopier/cmd.BuildTime=%BUILD_TIME_FULL%" -o %BUILD_DIR%\gopier-darwin-arm64
if errorlevel 1 echo   ✗ macOS ARM64 build failed

echo Cross-platform build complete
goto :end

:test
echo Running tests...
go test -v ./...
if errorlevel 1 (
    echo Test errors occurred
    exit /b 1
)
echo Tests complete
goto :end

:test-coverage
echo Running test coverage...
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
echo Coverage report: coverage.html
goto :end

:tidy
echo Tidying dependencies...
go mod tidy
go mod verify
echo Dependencies tidy complete
goto :end

:clean
echo Cleaning up...
if exist %BINARY_NAME% (
    del %BINARY_NAME%
    echo Deleted: %BINARY_NAME%
)
if exist %BUILD_DIR% (
    rmdir /s /q %BUILD_DIR%
    echo Deleted: %BUILD_DIR%
)
if exist coverage.out (
    del coverage.out
    echo Deleted: coverage.out
)
if exist coverage.html (
    del coverage.html
    echo Deleted: coverage.html
)
echo Cleanup complete
goto :end

:install
echo Installing...
if not exist %BINARY_NAME% (
    echo Error: Build file not found. Please build first.
    exit /b 1
)

set "GOPATH=%USERPROFILE%\go"
if not exist "%GOPATH%\bin" mkdir "%GOPATH%\bin"
copy %BINARY_NAME% "%GOPATH%\bin\" >nul
echo Installation complete: %GOPATH%\bin\%BINARY_NAME%
goto :end

:help
echo Gopier Build Script
echo ===================
echo.
echo Usage:
echo   build.bat [action]
echo.
echo Actions:
echo   build        - Normal build
echo   release      - Release build (optimized)
echo   cross-build  - Cross-platform build
echo   test         - Run tests
echo   test-coverage- Test coverage
echo   tidy         - Tidy dependencies
echo   clean        - Clean up
echo   install      - Install
echo   help         - Show this help
echo.
echo Examples:
echo   build.bat build
echo   build.bat test
echo   build.bat release
echo   build.bat clean
goto :end

:end
endlocal 