#!/bin/bash
set -euo pipefail
source /home/gearhead/Downloads/spank-linux/spank/parts/fuck-linux/run/environment.sh
set -x
go mod download all
go install -p "8"  ./...
