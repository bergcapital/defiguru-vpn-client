#!/usr/bin/env bash
set -euo pipefail

task_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
target="${1:-macos}"
build_dir="${BUILD_PATH:-$task_root/build-defiguru-$target}"
qt_root="${QT_ROOT_PATH:-}"
if [[ -z "$qt_root" ]]; then
    echo "Qt 6.10+ is required. Set QT_ROOT_PATH to the Qt version directory (containing macos or android_arm64_v8a)." >&2
    exit 2
fi
command -v cmake >/dev/null || { echo "CMake 3.25+ is required." >&2; exit 2; }
command -v conan >/dev/null || { echo "Conan 2.28 is required." >&2; exit 2; }

args=(-S "$task_root" -B "$build_dir" -DCMAKE_BUILD_TYPE=Release)
case "$target" in
    macos)
        args+=(-G Xcode -DMACOS_NE=ON -DCONAN_INSTALL_BUILD_CONFIGURATIONS=Release -DDEFIGURU_REVIEW_CAPTURE=ON
            "-DCMAKE_PREFIX_PATH=$qt_root/macos" -DCMAKE_XCODE_ATTRIBUTE_CODE_SIGNING_ALLOWED=NO
            -DCMAKE_XCODE_ATTRIBUTE_CODE_SIGNING_REQUIRED=NO)
        ;;
    android)
        : "${ANDROID_HOME:?Set ANDROID_HOME to the Android SDK directory}"
        : "${ANDROID_NDK_ROOT:?Set ANDROID_NDK_ROOT to Android NDK 27.0.11718014}"
        host_dir="$qt_root/gcc_64"
        [[ -d "$qt_root/macos" ]] && host_dir="$qt_root/macos"
        args+=(-G Ninja "-DCMAKE_TOOLCHAIN_FILE=$qt_root/android_arm64_v8a/lib/cmake/Qt6/qt.toolchain.cmake"
            "-DQT_HOST_PATH=$host_dir" "-DANDROID_SDK_ROOT=$ANDROID_HOME" "-DANDROID_NDK_ROOT=$ANDROID_NDK_ROOT"
            -DANDROID_ABI=arm64-v8a -DQT_ANDROID_ABIS=arm64-v8a -DAPP_ANDROID_MIN_SDK=30
            -DANDROID_PLATFORM=android-30 -DANDROID_BUILD_PLAY=OFF)
        ;;
    ios)
        args+=(-G Xcode "-DCMAKE_TOOLCHAIN_FILE=$qt_root/ios/lib/cmake/Qt6/qt.toolchain.cmake"
            "-DQT_HOST_PATH=$qt_root/macos" -DCMAKE_OSX_SYSROOT=iphoneos
            -DCONAN_INSTALL_BUILD_CONFIGURATIONS=Release -DCMAKE_XCODE_ATTRIBUTE_CODE_SIGNING_ALLOWED=NO
            -DCMAKE_XCODE_ATTRIBUTE_CODE_SIGNING_REQUIRED=NO)
        ;;
    *) echo "Usage: $0 [macos|android|ios]" >&2; exit 2 ;;
esac

# Unsigned review artifacts only. No signing identity, keychain or store upload.
cmake "${args[@]}"
cmake --build "$build_dir" --config Release --parallel "${JOBS:-3}"
if [[ "$target" == android ]]; then
    cmake --build "$build_dir" --config Release --target apk
fi
