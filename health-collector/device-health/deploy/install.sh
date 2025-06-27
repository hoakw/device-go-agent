#!/bin/bash

sudo mkdir -p /etc/sdt/device-health
sudo cp config.yaml /etc/sdt/deivce-health/
sudo cp device-health /usr/local/bin/
sudo cp device-health.service /etc/systemd/system/
sudo systemctl start device-health
sudo systemctl enable device-health
