# Security

## Reporting

请不要在公开 Issue 中粘贴 `pacmtoken`、Cookie、截图中的二维码或任何账号信息。安全相关问题请使用仓库的 GitHub Security Advisory 私下报告。

## Credential handling

- 咪咕登录使用官方登录页完成（应用内嵌登录页或扫码），应用不会读取或保存账号密码。
- 应用内登录页使用一次性网页会话：完成登录或关闭授权窗口后，网页 Cookie 立即清除。
- `pacmtoken` 只写入应用配置目录下的 `auth.json`，文件权限为 `0600`：
  - macOS: `~/Library/Application Support/MiguWave/auth.json`
  - Windows: `%APPDATA%\MiguWave\auth.json`
- 应用不读写 macOS 钥匙串或 Windows 凭据管理器，重新安装后不会触发系统授权弹窗。
- 环境变量 `MIGU_PACTOKEN` 和 `FREEWAVE_MIGU_TOKEN` 可用于临时调试，优先级高于本地令牌文件。
- 令牌不会写入 `state.json`、音频缓存、日志或界面回显。
- 「清除授权」会删除本地令牌文件。

## Local data

- macOS 配置和资料库：`~/Library/Application Support/MiguWave`
- macOS 音频缓存：`~/Library/Caches/MiguWave/audio`
- Windows 配置和资料库：`%APPDATA%\MiguWave`
- Windows 音频缓存：`%LOCALAPPDATA%\MiguWave\audio`
- 应用不包含遥测、崩溃上报或第三方分析 SDK。
