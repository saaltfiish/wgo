#! /bin/sh
#
# .example.sh
# Copyright (C) 2021 Odin <odinmanlee@gmail.com>
#
# Distributed under terms of the MIT license.
#

# `set -e`会导致任何一个命令失败就退出整个脚本
# set -e
set -x

export GOPROXY="https://mirrors.aliyun.com/goproxy/"
export GOSUMDB=off
export TZ="Asia/Shanghai"

prog="example"
version="1.0.0"

if [ -z $1 ]; then
    level="production"
else
    level=$1
fi

if [ -x "$(command -v git)" -a -d ".git" ]; then
    TagVersion=$(git describe --tags)
    if [ $? -eq 0 ]; then
        version=${TagVersion}
    else
        GITCOMMIT=$(git rev-parse HEAD)
    fi
else
	GITCOMMIT=$(date '+%Y%m%d%H%M%S')
fi

if [ "$GOOS" = "linux" ]; then
    go build -ldflags "-X wgo.AppVersion=${version} -X wgo.GitCommit=${GITCOMMIT} -X wgo.AppLevel=${level} -X 'wgo.BuildTime=`date -R`' -s -w -extldflags '-static'" -o release/${prog}
else
    go build -ldflags "-X wgo.AppVersion=${version} -X wgo.GitCommit=${GITCOMMIT} -X wgo.AppLevel=${level} -X 'wgo.BuildTime=`date -R`' -s -w" -o ${prog}
fi

exit
