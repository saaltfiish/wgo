#!/bin/sh
docker build --rm=true -f docker/Dockerfile.dev -t 127.0.0.1:5001/arch/wgo:modules .
docker push 127.0.0.1:5001/arch/wgo:modules
