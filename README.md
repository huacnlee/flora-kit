flora-kit
---------

基于 [shadowsocks-go](https://github.com/shadowsocks/shadowsocks-go) 做的完善实现，完全兼容 Surge 的配置文件。

> NOTE: 目前已完整实现自动翻墙的逻辑，可以用了。

## 功能列表

- ShadowSocks 代理，实现 socks 代理;
- 域名关键词、前缀、后缀白名单，黑名单；
- IP 白名单，黑名单；
- GeoIP 判断区域，自动选择线路；
- 自动改变 macOS 网路代理配置；

## TODO

- HTTP, HTTPS proxy 实现；
- 自动代理 pac 实现；

## 运行

下载 https://github.com/huacnlee/flora-kit/releases/download/0.1/flora-0.1.zip

由于启动的时候，需要修改系统的网络配置，所以你需要用 sudo 来执行:

```bash
cd flora
sudo ./flora-kit
```

## License

Apache License 2.0
