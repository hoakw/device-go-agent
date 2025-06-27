#!/bin/bash

sudo cp process-checker /usr/local/bin/
sudo cp process-checker.service /etc/systemd/system/
sudo systemctl start process-checker
sudo systemctl enable process-checker
