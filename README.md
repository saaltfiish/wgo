# WGO of gladsheim
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

可以放到`$GOPATH/src/wgo`, `vendor/wgo`


## 配置

支持文件配置(JSON/YAML)


### 支持双HTTP引擎(native HTTP & fasthttp)


### custom engine
