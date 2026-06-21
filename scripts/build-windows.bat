@echo off
setlocal

pushd "%~dp0.."
if errorlevel 1 exit /b %errorlevel%

if not exist "dist" mkdir "dist"
if errorlevel 1 (
    popd
    exit /b 1
)

set "CGO_ENABLED=0"
set "GOOS=windows"
set "GOARCH=amd64"

go build -ldflags="-s -w" -o "dist\imap-scrub-windows-amd64.exe" .
set "BUILD_EXIT=%errorlevel%"

popd
exit /b %BUILD_EXIT%
