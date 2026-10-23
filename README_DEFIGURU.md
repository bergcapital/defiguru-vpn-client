# DeFiGuru VPN

Native DeFiGuru VPN client based on the official AmneziaVPN `5.0.3.0` source.
Powered by [YLDX.AI](https://yldx.ai).

Public corresponding project source: [bergcapital/defiguru-vpn-client](https://github.com/bergcapital/defiguru-vpn-client).

This patch contains source changes and **unsigned review builds**. A successful build
does not establish App Store approval, notarization, production signing, or a
working tunnel on a customer's phone.

## Customer setup

1. Get your personal connection in [DeFiGuru VPN](https://t.me/DeFiGuru_VPN_bot).
2. Open DeFiGuru VPN, paste the connection and tap **Add my VPN**.
3. Confirm the imported connection and tap **Connect**. Allow the system VPN
   permission when requested.

The normal path needs no server address, SSH login, password, or manual protocol
fields. A settings file and QR import remain alternative options. Private
connection settings belong only to their recipient.

## Build

Requirements: Xcode for Apple platforms, Qt `6.10.1+` with Remote Objects,
Core5Compat, Shader Tools, Image Formats and the target platform kit; CMake
`3.25+`, Conan `2.28.0`, and sufficient build storage. Android additionally uses
JDK 17, SDK/build tools 36 and NDK `27.0.11718014`. Protocol dependencies are
pinned by upstream Conan recipes.

```sh
git submodule update --init --recursive
python3 branding/validate.py
export QT_ROOT_PATH=/path/to/Qt/6.10.1
bash branding/build-unsigned.sh macos
# or: bash branding/build-unsigned.sh ios
# or, after setting ANDROID_HOME and ANDROID_NDK_ROOT:
bash branding/build-unsigned.sh android
```

The manually triggered **DeFiGuru VPN unsigned review builds** workflow compiles
macOS and Android artifacts on standard hosted runners. It never signs, uploads
to a store, publishes a release, or reads customer credentials. Inherited
upstream deployment workflows are archived outside `.github/workflows`.

## Signing and distribution

- Apple application ID: `ai.defiguru.vpn`.
- Apple extension ID: `ai.defiguru.vpn.network-extension`.
- Shared group: `group.ai.defiguru.vpn`.
- Android installed application ID: `ai.defiguru.vpn`; the original Java/JNI
  namespace is intentionally retained.
- iOS needs the owner's organization Apple Developer team, app/extension IDs
  with Network Extensions and App Groups, corresponding provisioning profiles,
  signing certificates, and App Store Connect app record. The fork does not
  reuse the upstream team's identity or profiles.
- macOS needs owner signing/provisioning for the VPN extension and a signed,
  notarized distribution artifact before end-user release.
- Android production builds need an owner upload key/Play signing setup or a
  persistent APK signing key for direct distribution; an unsigned APK is not an
  installable customer release. Play review also requires its VPN declaration
  and an accurate privacy/data disclosure.
- Windows and Linux source remain upstream-compatible build targets; branded
  installers need their platform packaging, signing and tunnel checks before
  publication.

Store approval and regional availability are separate from compilation. Crypto
checkout remains on the website; no store checkout implementation or automatic
money transfer is added to the native client.

## Compatibility and licenses

The `vpn://` key is an import payload: base64url encoding of Qt's `qCompress`
format (four-byte big-endian uncompressed byte length, then zlib data). The
official parser accepts a full raw `.conf` after decompression as well as an
Amnezia JSON export. It is not evidence that an operating system registers
`vpn://` as an app-launch URL. Raw AWG fields must not be discarded or converted
to ordinary WireGuard parameters.

See [upstream provenance and distribution notes](branding/UPSTREAM.md), the
retained [GPLv3 license](LICENSE), and [original upstream README](README.md).
