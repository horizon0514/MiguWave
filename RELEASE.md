# Release

## 发布产物

每次发布至少包含：

- `miguwave-<version>.dmg`
- `miguwave-setup-<version>-amd64.exe`
- `miguwave-setup-<version>-arm64.exe`
- `SHA256SUMS`
- 对应版本的 `CHANGELOG.md` 条目
- 已知限制和 Gatekeeper 安装说明

## 前置检查

```bash
gofmt -l *.go
go test ./...
go vet ./...
```

`gofmt -l *.go` 必须没有输出。

## 构建

推荐使用脚本构建 macOS 与 Windows 产物：

```bash
./scripts/build-release.sh
```

脚本会依次执行格式检查、测试、静态检查，构建 `darwin/universal`、`windows/amd64` 和 `windows/arm64`，并生成 SHA-256 校验。

Windows 安装包需要 NSIS：

```bash
brew install nsis
```

只构建 Windows 时：

```bash
go tool mygo build -platform windows/amd64,windows/arm64
```

## GitHub Actions

推送到 `main` 会自动运行格式检查、测试、静态检查，并构建 macOS 与 Windows 产物。推送 `v*` 标签会自动创建 GitHub Release：

```bash
git tag -a v0.1.2 -m "MiguWave 0.1.2"
git push origin v0.1.2
```

Release 包含通用 DMG、macOS 应用 ZIP、Windows x64/ARM64 安装包与免安装版，以及合并后的 `SHA256SUMS`。工作流文件是 `.github/workflows/ci.yml`。

## 签名与公证

如果本机存在 `Developer ID Application` 证书：

```bash
security find-identity -v -p codesigning
xcrun notarytool store-credentials miguwave \
  --apple-id "你的 Apple ID" \
  --team-id "你的 Team ID" \
  --password "App 专用密码"
go tool mygo build -platform darwin/universal \
  -sign "Developer ID Application: Name (TEAMID)"
```

也可以在 `mygo.json` 中设置：

```json
{
  "macos": {
    "signingIdentity": "Developer ID Application: Name (TEAMID)",
    "notarize": {
      "keychainProfile": "miguwave"
    }
  }
}
```

不要把 Apple ID、专用密码、私钥或任何令牌写入仓库。

没有 Developer ID 证书时，MyGo 会生成 ad-hoc 签名包。它可以在构建机上运行，但其他 Mac 会触发 Gatekeeper；发布说明必须明确这一点，不能标记为「已签名」或「已公证」。

本机开发不要直接覆盖安装 ad-hoc 包。运行 `./scripts/build-local.sh --install` 会创建并复用一个仅限本机的自签名身份，避免每次重新构建后 macOS 把 App 当成新程序、反复触发系统安全提示。这个本机身份不能用于发布，也不要把 `~/Library/Application Support/MiguWave/local-signing` 分发给其他人。

Windows 安装包当前未做代码签名，SmartScreen 会提示未知发布者；发布说明需要给出「更多信息 → 仍要运行」的步骤。要签名时在 `mygo.json` 中配置证书：

```json
{
  "windows": {
    "certificate": "certs/code-signing.pfx"
  }
}
```

证书密码通过 `MYGO_WINDOWS_CERTIFICATE_PASSWORD` 提供，不要把 `.pfx` 或密码提交到仓库。

## 产物验证

```bash
app="build/darwin-universal/MiguWave.app"
dmg="build/darwin-universal/miguwave-0.1.2.dmg"

codesign --verify --deep --strict --verbose=2 "$app"
spctl -a -vv "$app"
shasum -a 256 -c SHA256SUMS
xcrun stapler validate "$dmg"
```

Windows 产物检查：

```bash
file build/windows-amd64/MiguWave.exe
file build/windows-arm64/MiguWave.exe
```

在 Windows 10/11 上分别安装一次，确认开始菜单快捷方式、卸载项、登录页和播放正常。

- Developer ID 发布：`spctl` 必须显示 `accepted`，`stapler` 必须通过。
- ad-hoc 发布：`codesign --verify` 通过即可；`spctl` 会显示 `rejected`，发布说明必须提供右键打开和解除隔离的步骤。

最后挂载 DMG，确认应用可以拖入「应用程序」，并至少完成一次启动、搜索、应用内登录和播放。

## 发布清单

- [ ] 版本号已更新
- [ ] `CHANGELOG.md` 已更新
- [ ] 格式、测试和静态检查通过
- [ ] 通用 DMG 构建成功
- [ ] Windows x64 / ARM64 安装包构建成功
- [ ] 签名状态已记录
- [ ] 已公证版本通过 `stapler validate`
- [ ] SHA-256 校验文件已生成
- [ ] DMG 已实际挂载检查
- [ ] Windows 安装、启动、应用内登录和播放已实际检查
- [ ] `MIGU_LIVE_TEST=1 go test -run 'TestLiveMigu' -v` 通过
- [ ] Release Notes 包含安装方式、已知限制和隐私说明
