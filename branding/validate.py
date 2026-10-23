#!/usr/bin/env python3
"""Validate release identities and private-import handling without SDK secrets."""
import json
from pathlib import Path
import plistlib
import xml.etree.ElementTree as ET

root = Path(__file__).resolve().parents[1]
for path in [root / 'client/ios/app/main.entitlements',
             root / 'client/ios/networkextension/AmneziaVPNNetworkExtension.entitlements',
             root / 'client/macos/app/app.entitlements',
             root / 'client/macos/networkextension/AmneziaVPNNetworkExtension.entitlements']:
    data = plistlib.loads(path.read_bytes())
    assert data['com.apple.security.application-groups'] == ['group.ai.defiguru.vpn'], path
    assert 'packet-tunnel-provider' in data['com.apple.developer.networking.networkextension'], path
for folder in [root / 'client/ios/app/Media.xcassets/AppIcon.appiconset',
               root / 'client/macos/app/Images.xcassets/AppIcon.appiconset']:
    for image in json.loads((folder / 'Contents.json').read_text())['images']:
        if 'filename' in image:
            assert (folder / image['filename']).is_file(), image
for path in root.glob('branding/*.qrc'):
    for item in ET.parse(path).getroot().iter('file'):
        assert (path.parent / item.text).is_file(), item.text
gradle = (root / 'client/android/build.gradle.kts').read_text()
assert 'applicationId = "ai.defiguru.vpn"' in gradle
assert 'namespace = "org.amnezia.vpn"' in gradle  # Native bridge remains compatible.
manifest = (root / 'client/android/AndroidManifest.xml').read_text()
assert 'android:authorities="${applicationId}.qtprovider"' in manifest
importer = (root / 'client/core/controllers/selfhosted/importController.cpp').read_text()
assert 'qDebug().noquote() << QJsonDocument(config).toJson()' not in importer
assert (root / 'LICENSE').read_text().startswith('                    GNU GENERAL PUBLIC LICENSE')
workflows = {p.name for p in (root / '.github/workflows').glob('*.yml')}
assert workflows == {'defiguru-native-review.yml', 'vpn-transport-canary.yml'}
for name in workflows:
    assert '  workflow_dispatch:' in (root / '.github/workflows' / name).read_text()
print('PASS: Apple identities and entitlements, Android identity/JNI isolation, resources, private-import logging, GPL, manual build boundary')
