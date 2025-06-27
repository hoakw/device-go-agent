#/bin/bash

GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o main.exe
mv main.exe device-heartbeat.exe
mv device-heartbeat.exe ../../sdt-cloud-deploy/v4/bwc-installer-win/bin
#cp device-heartbeat ../../deploy-bwc/heartbeat/
