#!/bin/sh
docker build --rm=true -f docker/Dockerfile -t 127.0.0.1:5001/arch/wgo .
docker push 127.0.0.1:5001/arch/wgo
