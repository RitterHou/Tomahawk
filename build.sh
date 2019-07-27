#!/usr/bin/env bash

flags="-s -w -X 'main.buildStamp=$(date '+%Y-%m-%d %H:%M:%S')' -X main.version=v1.0.0 -X 'main.goVersion=$(go version)'"
# echo "Build flags: ${flags}"

start_time=`date +%s`

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$flags" -o Tomahawk.exe
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "$flags" -o Tomahawk_Linux
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "$flags" -o Tomahawk_Darwin

end_time=`date +%s`
cost=$[$end_time-$start_time]
echo "Build success, cost ${cost} second(s)"
