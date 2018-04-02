# gladsheim WGO
---
go server framework

## docker

```shell
$ docker build --rm=true -f Dockerfile -t 127.0.0.1:5001/arch/wgo .
$ docker push 127.0.0.1:5001/arch/wgo
```

## import notes

import code:

```go
import "wgo"
```

可以放到$GOPATH/src/wgo, vendor/wgo


## 配置

支持文件配置(JSON/XML/YAML)以及uranus(JSON)

```json
{
    "dockerize": true,
    "daemonize": false,
    "enable_cache": true,
    "proc_name": "odintest1",
    "port": "9999",
    "version": "v0.9",
    "access": {
        "path": "/Users/Odin/dev/go/workspaces/misc/src/wgo_example/logs/access1.log",
        "topic": "wgo_access"
    },
    "proxy": {
        "*": {
            "/proxy": {
                "ttl": 10,
                "params": ["name"],
                "addrs": ["http://127.0.0.1:8888"]
            }
        }
    },
    "logs": [
        {
            "type": "console",
            "tag": "wgo",
            "format": "%T%E[%C] %M",
            "level": "DEBUG|INFO|WARNING|ERROR|FATAL"
        }
    ],
    "servers":[
        {
            "name": "odin",
            "addr": ":9999"
        },
        {
            "name": "odinrpc",
            "mode": "wrpc",
            "addr": ":50051"
        },
        {
            "name": "wepiao",
            "engine":"standard",
            "addr": ":9998"
        }
    ],
    "app":{
        "config1": "hi",
        "config2": "odin"
    },
    "app1": {
        "url": "http://sina.com.cn/"
    }
}
```

Directive   | Type   |Description
:-----------|:-------|:------------------
proc_name   | string | 
dockerize   | string | 
daemonize   | string | 
enable_cache| string | 
debug       | bool   | 
app_dir     | string | 
work_dir    | string | 
conf_dir    | string | 
time_zone   | string | 
access      | object | 
logs        | object | 
servers     | object | 
listen      | object | 

### 支持双HTTP引擎(native HTTP & fasthttp)



