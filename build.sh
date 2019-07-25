#!/usr/bin/env bash

flags="-X 'main.buildStamp=$(date '+%Y-%m-%d %H:%M:%S')' -X main.version=v1.0.0 -X 'main.goVersion=$(go version)'"
# echo "Build flags: ${flags}"

go build -ldflags "$flags" -o Tomahawk
