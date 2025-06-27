#/bin/bash

GOOS=linux GOARM=7 GOARCH=arm go build -o main main.go
mv main bwc
mv bwc ../sdt-cloud-deploy/v3/bwc-installer/bin/arm/
#cp device-control ../deploy-bwc-arm32/device-control/
