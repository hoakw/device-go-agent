#/bin/bash

GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o main.exe
mv main.exe process-checker.exe
mv process-checker.exe ../sdt-cloud-deploy/v4/bwc-installer-win/bin
#cp process-checker ../deploy-bwc/process-checker/
