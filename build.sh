#!/usr/bin/env bash

case $1 in
--release|-r)

    # 获取最新的tag
    git fetch --tags
    latestTag=$(git describe --tags `git rev-list --tags --max-count=1`)

    flags="-s -w -X 'main.buildStamp=$(date '+%Y-%m-%d %H:%M:%S')' -X main.version=${latestTag} -X 'main.goVersion=$(go version)'"
    echo "Build flags: ${flags}"

    start_time=`date +%s`

    # 所有目标的编译平台
    declare -a platforms=(
#        "android:arm"
        "darwin:386"
        "darwin:amd64"
#        "darwin:arm"
#        "darwin:arm64"
#        "dragonfly:amd64"
        "freebsd:386"
        "freebsd:amd64"
        "freebsd:arm"
        "linux:386"
        "linux:amd64"
        "linux:arm"
        "linux:arm64"
#        "linux:ppc64"
#        "linux:ppc64le"
#        "linux:mips"
#        "linux:mipsle"
#        "linux:mips64"
#        "linux:mips64le"
#        "netbsd:386"
#        "netbsd:amd64"
#        "netbsd:arm"
        "openbsd:386"
        "openbsd:amd64"
        "openbsd:arm"
#        "plan9:386"
#        "plan9:amd64"
        "solaris:amd64"
        "windows:386"
        "windows:amd64"
    )

    for platform in "${platforms[@]}"
    do
        IFS=':' read -ra p <<< "$platform"
        os=${p[0]}
        arch=${p[1]}
        # 执行编译
        CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build -ldflags "$flags" -o Tomahawk_${os}_${arch}
        echo Tomahawk_${os}_${arch}
    done

    end_time=`date +%s`
    cost=$[$end_time-$start_time]
    echo "Build success, cost ${cost} second(s)"
    ;;
*)
    go build
    ;;
esac
