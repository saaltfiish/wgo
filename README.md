# WGO of gladsheim
go server framework

## docker&go modules

为了缓存包, 创建wgo:basic镜像

```shell
$ docker/build_basic.sh
$ docker login --username=100005560250 ccr.ccs.tencentyun.com
$ docker tag 127.0.0.1:5001/arch/wgo:basic ccr.ccs.tencentyun.com/phyzi/wgo:basic
$ docker push ccr.ccs.tencentyun.com/phyzi/wgo:basic
```


## 配置

支持文件配置(JSON/YAML)


### 支持双HTTP引擎(native HTTP & fasthttp)


### custom engine
