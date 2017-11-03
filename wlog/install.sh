#!/bin/bash
DIR=`cd $(dirname $0); pwd`

if [ ! -n "$GOPATH" ]; then
    echo "GOPATH IS NULL"
else
	rm -rf $GOPATH/src/wepiao.com/logger
	mkdir -p $GOPATH/src/wepiao.com/
	cp -r logger $GOPATH/src/wepiao.com/logger
fi
