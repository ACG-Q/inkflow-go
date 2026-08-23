@echo off
cd /d "%~dp0"
echo [1/2] Building frontend...
cd frontend && call npm run build
if %errorlevel% neq 0 exit /b %errorlevel%
cd ..
echo [2/2] Building Go backend...
go build -ldflags="-s -w" -o dist\inkflow.exe .
echo Build complete: dist\inkflow.exe
