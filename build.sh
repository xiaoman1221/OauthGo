#!/usr/bin/env bash
set -e

echo "==> 构建前端..."
cd web
npm install
npm run build
cd ..

echo "==> 构建后端..."
mkdir -p bin
go build -o bin/oauthgo .

echo "==> 构建完成: ./bin/oauthgo"
