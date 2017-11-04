#!/bin/sh

set -e
set -x

if [ "$GOOS" = "linux" ]; then
    go install -ldflags '-extldflags "-static"' wgo
else
    go install wgo
fi
