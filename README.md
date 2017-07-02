Flora
-----

基于 [shadowsocks-go](https://github.com/shadowsocks/shadowsocks-go) 做的完善实现，完全兼容 Surge 的配置文件。

> NOTE: 目前已完整实现自动 Proxy 的逻辑，可以用了，已在自己的 macOS 环境连续跑了两天，稳定有效。

<img width="626" alt="2016-11-22 11 00 00" src="https://cloud.githubusercontent.com/assets/5518/20509326/d9a2ad9a-b0a2-11e6-9b9c-f6a59445b8d9.png">

## 功能列表

- macOS 和 Linux 同时支持；
- 连接 ShadowSocks 代理，并在本地建立 socks 代理服务，以提供给系统代理配置使用;
- 支持域名关键词、前缀、后缀匹配，制定 Direct 访问（白名单）或用 Proxy 访问（黑名单）；
- 支持 IP 白名单，黑名单；
- 支持 GeoIP 判断目标网站服务器所在区域，自动选择线路；
- 启动的时候自动改变 macOS,windows 网路代理配置，无需手工调整；


## TODO

- HTTP, HTTPS proxy 实现；
- 自动代理 pac 实现；
- 支持 Linux 网络代理自动设置;

## 下载 && 运行

https://github.com/huacnlee/flora-kit/releases

请根据系统下载需要的 release 包。

> NOTE: 由于启动的时候，需要修改系统的网络配置，所以你需要用 sudo 来执行:

#### macOS
```
$ cd flora
$ sudo ./flora
```

#### Linux
```
$ cd flora
$ ./flora
```

#### Windows
```
flora.exe
```

## License

Apache License 2.0
