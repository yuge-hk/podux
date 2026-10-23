<a name="readme-top"></a>

<!-- PROJECT SHIELDS -->

[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![Apache License][license-shield]][license-url]

<div align="right">
  <a href="README.md">English</a> | <b>简体中文</b>
</div>

> **项目更名通知：** 本项目原名为 **frpc-hub**，现已正式更名为 **Podux**。
> 仓库地址及所有相关引用均已同步更新。

<!-- PROJECT LOGO -->
<br />
<div align="center">
  <a href="https://github.com/luckjiawei/podux">
    <img src="site/src/assets/logo.png" alt="Logo" width="140">
  </a>

<h3 align="center">Podux</h3>

  <p align="center">
    <strong>Podux</strong> —— frpc 梦寐以求的 Web 前端管理面板。
    <br />
    在一个地方集中管理您所有的客户端、代理和连接。告别终端。无需再手动编辑配置文件。
    <br /><br />
    <a href="https://podux.io/guide/getting-started.html"><strong>🚀 快速开始</strong></a>
    &nbsp;&nbsp;·&nbsp;&nbsp;
    <a href="https://podux.io/"><strong>📖 官方文档</strong></a>
    &nbsp;&nbsp;·&nbsp;&nbsp;
    <a href="https://github.com/luckjiawei/podux/issues">🐛 提交 Bug</a>
    &nbsp;&nbsp;·&nbsp;&nbsp;
    <a href="https://github.com/luckjiawei/podux/issues">✨ 需求建议</a>
  </p>

  <br />

  <p align="center">
    <strong>✅ 多节点管理</strong> &nbsp;·&nbsp;
    <strong>📊 实时数据大盘</strong> &nbsp;·&nbsp;
    <strong>🌐 网络监控</strong> &nbsp;·&nbsp;
    <strong>🚀 一键开机自启</strong> &nbsp;·&nbsp;
    <strong>⚡ 高性能稳定</strong> &nbsp;·&nbsp;
    <strong>🐳 完美适配 Docker</strong> &nbsp;·&nbsp;
    <strong>🎨 现代化 UI 设计</strong> &nbsp;·&nbsp;
  </p>

</div>

## Cloudflare Tunnel 本地连接器

在现有登录后的 WebUI 中管理多个已有 Tunnel Token 对应的本机连接器，支持独立
启动、停止、重启、加密 Token 更新、自动启动/重试、回环 Metrics 和安全运行事件。

需单独安装 cloudflared（>= 2025.4.0，并通过能力检测），并将外置加密主密钥与数据
配套备份。标准 Docker 镜像不内置 cloudflared。该功能不管理 Cloudflare 账号/API、
DNS、Ingress 或 Access。参见[中文使用指南](docs/guide/cloudflare-tunnels.md)与
[部署及安全说明](docs/api/cloudflared.md)。

## 界面截图

![界面截图](screenshot/1.png)

## 版本里程碑

- 2026-03-03: 发布 v0.1.0
- 2026-03-12: 发布 v0.1.1
- 2026-03-14: 发布 v0.1.2
- 2026-03-16: 发布 v0.1.3
- 2026-03-25: 发布 v0.1.4
- 2026-03-26: 发布 v0.1.5
- 2026-03-31: 发布 v0.1.6

## License

[Apache 2.0](LICENSE)

## Star 历史

[![Star History Chart](https://api.star-history.com/svg?repos=luckjiawei/podux&type=Date)](https://www.star-history.com/#luckjiawei/podux&Date)

<!-- MARKDOWN LINKS & IMAGES -->

[forks-shield]: https://img.shields.io/github/forks/luckjiawei/podux.svg?style=for-the-badge
[forks-url]: https://github.com/luckjiawei/podux/network/members
[stars-shield]: https://img.shields.io/github/stars/luckjiawei/podux.svg?style=for-the-badge
[stars-url]: https://github.com/luckjiawei/podux/stargazers
[issues-shield]: https://img.shields.io/github/issues/luckjiawei/podux.svg?style=for-the-badge
[issues-url]: https://github.com/luckjiawei/podux/issues
[license-shield]: https://img.shields.io/github/license/luckjiawei/podux.svg?style=for-the-badge
[license-url]: https://github.com/luckjiawei/podux/blob/main/LICENSE
