#/bin/bash

GOOS=linux GOARM=7 GOARCH=arm go build -o main
#GOOS=linux GOARCH=arm64  go build -o main
mv main process-checker
mv process-checker ../sdt-cloud-deploy/v4/bwc-installer/bin/arm/
