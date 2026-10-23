#!/usr/bin/env python3
"""Mechanical icon sizes from the existing DeFiGuru brain artwork, on macOS."""
import json
from pathlib import Path
import plistlib
import subprocess

root = Path(__file__).resolve().parents[1]
source = root / 'branding/neural-brain.webp'

def render(destination, width, height=None):
    height = height or width
    subprocess.run(['sips', '-s', 'format', 'png', '--resampleHeightWidthMax', str(min(width, height)),
                    '--padColor', '101425', '--padToHeightWidth', str(height), str(width),
                    str(source), '--out', str(destination)], check=True, capture_output=True)

for folder in [root / 'client/ios/app/Media.xcassets/AppIcon.appiconset',
               root / 'client/macos/app/Images.xcassets/AppIcon.appiconset']:
    for item in json.loads((folder / 'Contents.json').read_text())['images']:
        if 'filename' in item:
            side = round(float(item['size'].split('x')[0]) * float(item['scale'].rstrip('x')))
            render(folder / item['filename'], side)

render(root / 'client/ios/app/launch.png', 1024)
for path in (root / 'client/android/res').glob('mipmap-*/*.png'):
    if path.name not in ['icon.png', 'icon_round.png', 'ic_launcher_foreground.png', 'ic_banner.png']:
        continue
    metadata = subprocess.check_output(['sips', '-g', 'pixelWidth', '-g', 'pixelHeight', str(path)], text=True)
    dimensions = [int(line.split(':')[1]) for line in metadata.splitlines() if ': ' in line]
    render(path, *dimensions)
