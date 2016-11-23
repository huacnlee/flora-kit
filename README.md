flora-kit
---------

基于 [shadowsocks-go](https://github.com/shadowsocks/shadowsocks-go) 做的完善实现，完全兼容 Surge 的配置文件。

> NOTE: 目前已完整实现自动 Proxy 的逻辑，可以用了，已在自己的 macOS 环境连续跑了两天，稳定有效。

<img width="626" alt="2016-11-22 11 00 00" src="https://cloud.githubusercontent.com/assets/5518/20509326/d9a2ad9a-b0a2-11e6-9b9c-f6a59445b8d9.png">

## 功能列表

- macOS 和 Linux 同时支持；
- ShadowSocks 代理，实现 socks 代理;
- 域名关键词、前缀、后缀白名单，黑名单；
- IP 白名单，黑名单；
- GeoIP 判断区域，自动选择线路；
- 自动改变 macOS 网路代理配置；

## TODO

- HTTP, HTTPS proxy 实现；
- 自动代理 pac 实现；
- 支持 Linux 网络代理自动设置;

## 下载 && 运行

https://github.com/huacnlee/flora-kit/releases

请根据系统下载需要的 release 包。

> NOTE: 由于启动的时候，需要修改系统的网络配置，所以你需要用 sudo 来执行:

```bash
cd flora
sudo ./flora-kit
```

## License

Apache License 2.0
