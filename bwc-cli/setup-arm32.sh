#/bin/bash

GOOS=linux GOARM=7 GOARCH=arm go build -o main main.go
mv main bwc
mv bwc ../sdt-cloud-deploy/v2/bwc-installer-arm32/
#cp device-control ../deploy-bwc-arm32/device-control/
