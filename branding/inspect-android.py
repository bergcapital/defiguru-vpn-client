#!/usr/bin/env python3
"""Inspect a built public APK; never handles signing keys or connects a VPN."""
import hashlib
import io
import json
from pathlib import Path
import re
import subprocess
import sys
from zipfile import ZipFile
from PIL import Image

root = Path(__file__).resolve().parents[1]
artifact, output, tools = map(Path, sys.argv[1:4])
expected_package = sys.argv[4] if len(sys.argv) > 4 else 'ai.defiguru.vpn'
output.mkdir(parents=True, exist_ok=True)
apks = sorted(artifact.rglob('*.apk'))
assert apks, 'No APK in the existing review artifact'
reference = Image.open(root / 'client/android/res/mipmap-xxxhdpi/ic_launcher_foreground.png').convert('RGBA')
reports = []
for index, apk in enumerate(apks):
    badging = subprocess.check_output([str(tools / 'aapt'), 'dump', 'badging', str(apk)], text=True)
    manifest = subprocess.check_output([str(tools / 'aapt'), 'dump', 'xmltree', str(apk), 'AndroidManifest.xml'], text=True)
    package = re.search(r"package: name='([^']+)'", badging).group(1)
    version = re.search(r"versionName='([^']+)'", badging).group(1)
    label = re.search(r"application-label:'([^']+)'", badging).group(1)
    assert package == expected_package, 'Unexpected installed package'
    expected_labels = {'DeFiGuru VPN Preview'} if package.endswith('.review') else {'DeFiGuruVPN', 'DeFiGuru VPN'}
    assert label in expected_labels, 'Unexpected application brand'
    assert expected_package + '.qtprovider' in manifest, 'Unexpected FileProvider authority'
    matched = None
    with ZipFile(apk) as archive:
        for name in archive.namelist():
            # Release resource shrinking can rename Android resource paths.
            # Compare decoded pixels rather than assuming the source filename.
            if name.endswith('.png'):
                raw = archive.read(name)
                actual = Image.open(io.BytesIO(raw)).convert('RGBA')
                if actual.size == reference.size and actual.tobytes() == reference.tobytes():
                    matched = name
                    (output / f'apk-{index + 1}-brand-icon.png').write_bytes(raw)
                    break
    assert matched, 'APK does not contain the audited DeFiGuru brain icon pixels'
    signature = subprocess.run([str(tools / 'apksigner'), 'verify', '--verbose', str(apk)], capture_output=True)
    if package.endswith('.review'):
        assert signature.returncode == 0, 'Preview must have a valid test signature'
    report = {'apk': apk.name, 'bytes': apk.stat().st_size,
              'sha256': hashlib.sha256(apk.read_bytes()).hexdigest(),
              'package': package, 'version': version, 'label': label,
              'providerAuthority': expected_package + '.qtprovider',
              'brandIconPixelMatch': True, 'brandIconResource': matched,
              'signature': 'signed' if signature.returncode == 0 else 'unsigned',
              'ownerReleaseSignatureProven': False, 'installOrTunnelTestPerformed': False}
    reports.append(report)
    print(json.dumps(report))
(output / 'apk-inspection.json').write_text(json.dumps({'apks': reports}, indent=2) + '\n')
