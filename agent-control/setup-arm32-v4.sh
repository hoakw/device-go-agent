#/bin/bash

GOOS=linux GOARM=7 GOARCH=arm go build -o main
#GOOS=linux GOARCH=arm64  go build -o main
mv main device-control
mv device-control ../sdt-cloud-deploy/v3/bwc-installer/bin/arm/
