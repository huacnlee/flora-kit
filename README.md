# flora-kit
Flora Proxy server

## 网络调整命令

列出所有网络链接方式

```
$ networksetup -listallnetworkservices
```

并循环设置

获取某个连接的 Socks Proxy 配置

```
$ networksetup -setsocksfirewallproxy <networkservice> <domain> <port number> <authenticated> <username> <password>
```

获取某个连接的 HTTP Procx 配置

```
$ networksetup -setwebproxy <networkservice> <domain> <port number> <authenticated> <username> <password>
```

获取某个连接的 HTTPS Proxy 配置

```
$ networksetup -setsecurewebproxy <networkservice> <domain> <port number> <authenticated> <username> <password>
```

设置 Proxy ByPass

```
$ networksetup -setproxybypassdomains <networkservice> <domain1> [domain2] [...]
```