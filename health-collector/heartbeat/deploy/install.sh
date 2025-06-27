#!/bin/bash

sudo mkdir -p /etc/sdt/device-heartbeat
sudo cp device-heartbeat /usr/local/bin/
sudo cp device-heartbeat.service /etc/systemd/system/
sudo systemctl start device-heartbeat
sudo systemctl enable device-heartbeat
