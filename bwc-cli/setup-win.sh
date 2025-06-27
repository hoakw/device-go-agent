#/bin/bash

GOOS=windows CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -o main.exe main-win.go
mv main.exe bwc.exe
mv bwc.exe ../gitea-ec2/bwc-installer-win/
#cp device-control ../deploy-bwc/device-control/
