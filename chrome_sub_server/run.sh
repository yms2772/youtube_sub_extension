#!/bin/bash
echo "Builing wasm..."
GOARCH=wasm GOOS=js go build -v -o ../chrome_sub_server_wasm/web/app.wasm
echo "Builing app..."
cd ../chrome_sub_server_wasm && go build -v