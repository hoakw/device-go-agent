#!/bin/bash

sudo mkdir -p /etc/sdt/device-control
sudo cp config.yaml /etc/sdt/device-control
sudo cp device-control /usr/local/bin/
sudo cp device-control.service /etc/systemd/system/
sudo systemctl start device-control
sudo systemctl enable device-control