#/bin/bash

GOOS=linux GOARM=7 GOARCH=arm go build -o main main.go
mv main bwc-init
mv bwc-init ../sdt-cloud-deploy/v4/bwc-installer/bin/arm/
#cp device-control ../deploy-bwc-arm32/device-control/
