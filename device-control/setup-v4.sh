#/bin/bash

GOOS=linux GOARCH=amd64 go build -o main main.go
mv main device-control
mv device-control ../sdt-cloud-deploy/v4/bwc-installer/bin/amd/
#cp device-control ../deploy-bwc/device-control/
