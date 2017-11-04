#!/bin/sh

set -e
set -x

if [ "$GOOS" == "linux" ]; then
    go build -ldflags '-extldflags "-static"' wgo
else
    go build wgo
fi
