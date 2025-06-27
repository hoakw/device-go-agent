#/bin/bash

GOOS=linux GOARCH=amd64 go build -o main
mv main bwc-management
mv bwc-management ../sdt-cloud-deploy/v4/bwc-installer/bin/amd/
#cp process-checker ../deploy-bwc/process-checker/
