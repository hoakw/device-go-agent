#/bin/bash

GOOS=linux GOARM=7 GOARCH=arm go build -o main
#GOOS=linux GOARCH=arm64  go build -o main
mv main device-health
mv device-health ../../sdt-cloud-deploy/v4/bwc-installer/bin/arm/
