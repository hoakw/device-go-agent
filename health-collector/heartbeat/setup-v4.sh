#/bin/bash

GOOS=linux GOARCH=amd64 go build -o main
mv main device-heartbeat
mv device-heartbeat ../../sdt-cloud-deploy/v4/bwc-installer/bin/amd/
#cp device-heartbeat ../../deploy-bwc/heartbeat/
