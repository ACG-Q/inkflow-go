#!/bin/bash
set -e
echo "[1/2] Building frontend..."
cd frontend && npm run build && cd ..
echo "[2/2] Building Go backend..."
go build -ldflags="-s -w" -o dist/inkflow .
echo "Build complete: dist/inkflow"
