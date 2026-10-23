# Upstream and distribution

DeFiGuru VPN is an independent branded fork of the official AmneziaVPN client.
It is not an Amnezia service or an endorsement by the Amnezia project.

- Base: `amnezia-vpn/amnezia-client`, stable tag `5.0.3.0`.
- Pinned commit: `de93650a90739b87bb47a632872ea9d0adc9412f`.
- Evaluated alternative: `amnezia-vpn/DefaultVPN`, dev
  `3cee753b9fb3658ddebdc0982036979e08c7d8ba` (application version `1.0.11.0`).
- The newer official base is used because its protocol dependencies include
  AmneziaWG Apple `3.1.4` and AWG Go/Android `3.1.20260814`, as well as maintained
  branding hooks. The DefaultVPN repository snapshot predates this support.

The original GPLv3 license and source copyright notices are retained. When
distributing binaries, provide the corresponding complete source for that exact
build, including these modifications and build scripts, under the GPL, and the
required notices/licenses for the dependency versions actually shipped. Qt,
OpenSSL, libssh, OpenVPN, WireGuard/AmneziaWG, Xray and other dependencies retain
their respective licenses. The Conan recipes and pinned submodule revisions are
the dependency inventory; an application build is not a substitute for bundling
the dependency license texts.

Names in protocol formats, C++ APIs, Java namespaces, and JNI bridges remain
upstream for compatibility. The installed application, settings storage,
keychain, bundle ID, app group, Android application ID, user-facing brand,
support contacts and source distribution belong to DeFiGuru.

No upstream service keys, signing identity, provisioning profile, purchase
account, private configuration, or customer record is included in this fork.
