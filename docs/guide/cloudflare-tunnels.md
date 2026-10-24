# Cloudflare Tunnel 本地管理

在登录后的主导航选择 **服务器 → Cloudflare Tunnel**，与 FRPC 共用服务器管理页。
旧 `/tunnels` 链接会跳转到对应标签。此页面管理已有 Tunnel 的本机
cloudflared 连接器，不创建云端 Tunnel，也不管理账号、DNS、Ingress 或 Access。

## 准备

在 Podux 服务器安装 cloudflared 2025.4.0 或更高版本，再重启 Podux。页面会显示
实际检测的版本、构建信息、Token File 和参数能力。未检测到程序时，仍可新建、
查看和编辑配置，但启动及重启按钮不可用。WebUI 不上传或下载可执行程序。

[后端部署、安全存储与密钥备份说明](../api/cloudflared.md) 中的外置主密钥必须
与数据备份配套保存。仅备份 PocketBase 数据库不足以恢复已加密 Token。

## 安装与运行要求

从 [Cloudflare 官方下载页](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/downloads/)
选择与 Podux 所在操作系统和 CPU 匹配的程序。最低版本为 2025.4.0；本功能还会检测
`--token-file` 及参数解析能力，不能只根据版本号判断。原生 CI 已验证 Linux amd64/arm64、
Windows amd64 和 macOS arm64 的本地运行路径，官方测试二进制固定为 2026.10.3。
真实账号连接与公网源站访问需要另行验收，不能由无效 Token 测试推断。

程序须对 Podux 的运行账号可执行；数据目录和外置密钥目录须可写、可持久化。
确保主机允许所选协议到 Cloudflare Edge 的出站连接。二进制路径只能由服务器管理员配置：

```sh
PODUX_CLOUDFLARED_BINARY=/usr/local/bin/cloudflared ./podux serve --dir /path/to/pb_data
```

Windows PowerShell 示例（请替换为实际安装路径）：

```powershell
$env:PODUX_CLOUDFLARED_BINARY = 'C:\Tools\cloudflared.exe'
.\podux.exe serve --dir 'C:\Podux\data'
```

标准 Podux Docker 镜像不包含 cloudflared。使用容器时需自行提供可在该容器内运行的
匹配程序，并持久化数据目录及其外置密钥；只挂载 `pb_data` 会漏掉默认的同级密钥目录。
可通过 `PODUX_CLOUDFLARED_KEY_FILE` 指定容器内独立、持久化的密钥文件位置。
Windows 当前要求自定义密钥路径与数据目录位于同一卷；跨卷路径会安全拒绝。
不要把 Token 写进命令行、镜像或部署清单；在现有认证保护的页面中输入，远程访问使用 HTTPS。

## 新建和编辑

点击“新建 Tunnel”，填写名称及已有 Tunnel 的运行 Token。多个配置可独立运行。
编辑时 Token 输入框始终为空；留空保留原值，明确输入新的 Token 才替换。
已有 Token 无法回显，也不会写入浏览器 localStorage、sessionStorage 或 URL。

自动启动表示 Podux 下次启动时启动此配置；异常自动重启只处理非人工异常退出。
最大额外重启次数为 0 时不追加重试，不代表无限重启。

高级区域支持协议、区域（默认/美国）、Edge IP 版本、绑定 IP、强制后量子、
连接重试、优雅退出秒数和 Metrics 端口。强制后量子需要显式选择 QUIC。
绑定 IP 必须与指定版本一致。Metrics 端口为 0 时自动分配，监听始终限于本地回环。
额外参数只提供后端白名单内的 Info 日志级别及 1–4 条 HA 连接；没有 TLS 跳过开关。

保存使用 revision 检查。发生冲突时保留未保存输入，不自动覆盖；确认“重新加载配置”
会丢弃当前草稿及新输入的 Token。运行中保存成功后，可选择“现在重启”或“稍后重启”。
保存本身不会触发自动重启；旧运行快照也用于既有监督器的自动重试。

## 状态和日志

启动返回“已接受”不等于连接成功。页面分别显示已停止、启动中、运行中、等待重试、
停止中、失败；实际协议只使用后端观测值，无法观测时显示“未知”。
连接器就绪不证明公网路由和源站已经端到端可达。

列表约每 5 秒刷新，切到后台暂停。详情显示连接数、运行代次、运行与保存配置版本、
重试信息及安全错误。详情打开时运行事件约每 3 秒增量读取，可暂停自动滚动、手动刷新
和复制安全日志。这里只保留最多 500 条脱敏事件，不提供原始 stdout/stderr 下载。
游标缺口会提示截断；日志 epoch 变化时重新读取（兼容旧服务器的游标回退检测）。
Podux 重启后事件可能消失。

删除须确认，会先停止进程，再永久移除配置及加密 Token；停止失败或 revision 冲突
不会从列表直接移除记录。

本机进程清理与 macOS/Windows 验证边界参见[后端运行说明](../api/cloudflared.md)。

## 界面预览

以下截图来自隔离环境的测试连接器，仅展示界面与进程生命周期，不代表真实 Edge 验收通过。

![桌面深色中文界面](/screenshots/cloudflare-tunnels-desktop.png)

<img src="/screenshots/cloudflare-tunnels-mobile.png" alt="移动端独立连接器管理" width="390" />

## 验证边界

正常启停、进程回收、凭证权限和运行器测试已有原生 CI 记录。macOS/Windows 在父进程
强制崩溃时没有通用的防孤儿进程保证；Windows 优雅信号为尽力执行，超时后终止直接子进程。
运行事件是有界、非持久的安全事件，不能替代原始 cloudflared 日志。Token 轮换需显式重启
运行实例；主密钥轮换尚未实现。不要将“已接受启动”或“连接器就绪”当成公网源站可达的证明。

本轮没有授权的非生产凭证时，真实 Edge/公网访问验收为 **Blocked**。最新测试与已知限制见
[Phase 4 验收报告](https://github.com/yuge-hk/podux/blob/feat/local-cloudflare-tunnels/development/cloudflared-phase4-validation.md)。
