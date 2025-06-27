#/bin/bash

GOOS=linux GOARCH=amd64 go build -o main
mv main device-health
mv device-health ../../sdt-cloud-deploy/v4/bwc-installer/bin/amd/
#cp device-health ../../deploy-bwc/device-health/
