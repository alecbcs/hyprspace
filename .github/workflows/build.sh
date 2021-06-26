#!/bin/bash

platforms=("linux/amd64" "linux/arm" "linux/arm64" "darwin/amd64" "darwin/arm64")

for platform in "${platforms[@]}"
do
    platform_split=(${platform//\// })
    GOOS=${platform_split[0]}
    GOARCH=${platform_split[1]}
    env GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build -ldflags "-s -w -X github.com/hyprspace/hyprspace/cli.appVersion=$1" -o hyprspace-$1-${GOOS}-${GOARCH} .

done