#!/bin/sh
set -eu
CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o leafmark .
