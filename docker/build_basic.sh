#! /bin/sh
#
# build_basic.sh
# Copyright (C) 2019 Odin <Odin@Odin-Pro.local>
#
# Distributed under terms of the MIT license.
#
# 用于建立modules缓存,  制作基础的wgo容器
# 只有当go.mod发生变化时, 这个基础容器需要更新
# 首先build在本地, 手动push到发布服务器, 范例:
# `docker tag 127.0.0.1:5001/arch/wgo:basic ccr.ccs.tencentyun.com/phyzi/wgo:basic`
# `docker push ccr.ccs.tencentyun.com/phyzi/wgo:basic`

docker build --rm=true -f docker/Dockerfile.basic -t 127.0.0.1:5001/arch/wgo:basic .

docker push 127.0.0.1:5001/arch/wgo:basic
