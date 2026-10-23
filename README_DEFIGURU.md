# DeFiGuru VPN

Native DeFiGuru VPN client based on the official AmneziaVPN `5.0.3.0` source.
Powered by [YLDX.AI](https://yldx.ai).

Public corresponding project source: [bergcapital/defiguru-vpn-client](https://github.com/bergcapital/defiguru-vpn-client).

This patch contains source changes and **unsigned review builds**. A successful build
does not establish App Store approval, notarization, production signing, or a
working tunnel on a customer's phone.

## Customer setup

1. Get your personal connection in [DeFiGuru VPN](https://t.me/DeFiGuru_VPN_bot).
2. Open DeFiGuru VPN and tap **Paste** to review your connection.
3. Tap **Connect**. Allow the system VPN
   permission when requested.

The explicit Paste button opens the review directly. **Add my VPN** remains a
fallback when pasting with the system keyboard. Confirmation keeps upstream
untrusted-configuration warnings and never interrupts an existing connection.

The welcome screen includes a language selector. The DeFiGuru setup and support
copy is translated into Russian, Italian and Spanish. Upstream Russian/Spanish
catalogs are retained. Italian also covers the primary import preview, connection
states, common errors/retry messages, home screen and application language settings.
Advanced server administration screens may fall back to English.

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
macOS, iOS and Android artifacts on standard hosted runners. Apple/release
artifacts are unsigned; the separate Android Preview uses a disposable debug
certificate. It never uploads to a store, publishes a release, or reads customer
credentials. Inherited
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

### Installable Android Preview

The review workflow also builds an OSS debug APK named
`DeFiGuruVPN-Android-Preview.apk`. It has package `ai.defiguru.vpn.review`, label
**DeFiGuru VPN Preview**, and the standard disposable Android debug certificate.
It installs alongside the future production package. This is a test build, not
a Play Store release or an owner-signed production APK. A later preview from a
different runner may require uninstalling the previous preview because debug
certificates are temporary; keep your personal connection available to reimport.
The workflow verifies its package, FileProvider authority, signature and actual
brand icon pixels. Release builds retain `ai.defiguru.vpn` and require owner signing.

### Minimal owner steps for iPhone distribution

1. Start an **Organization** enrollment at
   [Apple Developer Program](https://developer.apple.com/programs/enroll/).
   Prepare the legal entity name/address, D-U-N-S number, an authorized account
   holder, an Apple Account with two-factor authentication, organization-domain
   work email/phone and a working public company website.
2. The owner completes Apple's verification, accepts the agreement and pays the
   membership fee shown by Apple (currently USD 99/year, with regional pricing).
3. In the owner team, register the two app identifiers above and one shared app
   group. Enable Network Extensions and App Groups for the app/extension; create
   their development/distribution provisioning profiles and the App Store
   Connect app record. Signing stays in the owner's Apple/Xcode account.
4. Rebuild with the public Team ID and owner provisioning/signing, then upload
   the signed iPhone app to TestFlight for device testing and Apple review.

No enrollment, agreement acceptance, membership purchase, certificate creation
or store submission is performed by this source/build workflow. Do not post
passwords, private signing keys, certificates or provisioning secrets in chat.

After signing, physical iPhone/Android acceptance must check DNS, IPv4 and IPv6
routing/leaks, reconnect, background/resume and network switching. The independent
userspace DNS/HTTPS IPv4 canary proves the server transport and key format, not
native OS routes or phone leak protection. Store readiness requires those device
results in addition to compilation and signing.

### Actual welcome preview

The macOS review workflow captures the real welcome QML into a small PNG using
an isolated empty settings directory and no encrypted account/keychain. The
capture hook is compiled only with `DEFIGURU_REVIEW_CAPTURE=ON`; ordinary owner
release configuration defaults to OFF. It never imports a profile, presses
Connect or claims to test a native tunnel.

## Compatibility and licenses

The `vpn://` key is an import payload: base64url encoding of Qt's `qCompress`
format (four-byte big-endian uncompressed byte length, then zlib data). The
official parser accepts a full raw `.conf` after decompression as well as an
Amnezia JSON export. It is not evidence that an operating system registers
`vpn://` as an app-launch URL. Raw AWG fields must not be discarded or converted
to ordinary WireGuard parameters.

See [upstream provenance and distribution notes](branding/UPSTREAM.md), the
retained [GPLv3 license](LICENSE), and [original upstream README](README.md).
