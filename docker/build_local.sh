#! /bin/sh
#
# build_local.sh
# Copyright (C) 2019 Odin <Odin@Odin-Pro.local>
#
# Distributed under terms of the MIT license.
#
# 在wgo:basic的基础上, 用于建立wgo的本地image
# 本地测试用, 无需push

docker build --rm=true -f docker/Dockerfile.dev -t 127.0.0.1:5001/arch/wgo:latest .

docker push 127.0.0.1:5001/arch/wgo:latest
