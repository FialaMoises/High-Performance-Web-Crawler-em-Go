@echo off
REM Quick run script for Go Web Crawler (Windows)
REM Usage: run.bat [URL] [optional: depth] [optional: pages]

SETLOCAL

REM Default values
SET "URL=%~1"
IF "%URL%"=="" SET "URL=https://books.toscrape.com"

SET "DEPTH=%~2"
IF "%DEPTH%"=="" SET "DEPTH=3"

SET "PAGES=%~3"
IF "%PAGES%"=="" SET "PAGES=100"

SET "WORKERS=%~4"
IF "%WORKERS%"=="" SET "WORKERS=10"

echo ==============================================
echo   Go Web Crawler - Quick Run
echo ==============================================
echo URL:     %URL%
echo Depth:   %DEPTH%
echo Pages:   %PAGES%
echo Workers: %WORKERS%
echo ==============================================
echo.

REM Create output directory if it doesn't exist
IF NOT EXIST output mkdir output

REM Check if Docker is available
docker --version >nul 2>&1
IF %ERRORLEVEL% EQU 0 (
    echo 🐳 Running with Docker...

    REM Check if image exists
    docker images go-web-crawler:latest -q >nul 2>&1
    IF %ERRORLEVEL% NEQ 0 (
        echo 📦 Building Docker image...
        docker build -t go-web-crawler:latest .
    )

    REM Run crawler
    docker run -v "%CD%\output:/app/output" go-web-crawler:latest ^
        -url "%URL%" ^
        -depth %DEPTH% ^
        -pages %PAGES% ^
        -workers %WORKERS% ^
        -format both ^
        -log-level info

) ELSE (
    REM Check if Go is available
    go version >nul 2>&1
    IF %ERRORLEVEL% EQU 0 (
        echo 🔧 Running with Go...

        REM Download dependencies if needed
        IF NOT EXIST vendor (
            echo 📥 Downloading dependencies...
            go mod download
        )

        REM Run crawler
        go run ./cmd/crawler ^
            -url "%URL%" ^
            -depth %DEPTH% ^
            -pages %PAGES% ^
            -workers %WORKERS% ^
            -format both ^
            -log-level info
    ) ELSE (
        echo ❌ Error: Neither Docker nor Go found!
        echo Please install Docker or Go to run the crawler.
        exit /b 1
    )
)

echo.
echo ✅ Done! Check the output directory for results.

ENDLOCAL