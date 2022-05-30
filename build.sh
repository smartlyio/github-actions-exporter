#!/usr/bin/env bash

set -x

CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-X 'main.version=$VERSION'"  -o bin/app
