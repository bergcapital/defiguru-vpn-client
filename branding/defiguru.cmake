# DeFiGuru application identity. Protocol and Java/JNI namespaces stay upstream.
set(CLIENT_TARGET_NAME "DeFiGuruVPN" CACHE STRING "Application target")
set(CLIENT_APPLICATION_NAME "DeFiGuruVPN" CACHE STRING "Application executable")
set(CLIENT_SERVICE_NAME "DeFiGuruVPN-service" CACHE STRING "Privileged service")
set(CLIENT_ORGANIZATION_NAME "DeFiGuru" CACHE STRING "Settings organization")
set(CLIENT_APP_INSTANCE_NAME "DeFiGuruVPNInstance" CACHE STRING "Local instance")
set(CLIENT_KEYCHAIN_NAME "DeFiGuruVPN-Keychain" CACHE STRING "Encrypted settings keychain")
set(CLIENT_SWIFT_MODULE_NAME "DeFiGuruVPN" CACHE STRING "Swift module")
set(CLIENT_SWIFT_OBJC_HEADER_NAME "DeFiGuruVPN-Swift.h" CACHE STRING "Swift bridge")
set(CLIENT_NETWORK_EXTENSION_NAME "DeFiGuruVPNNetworkExtension" CACHE STRING "VPN extension")
set(CLIENT_NOTIFICATION_IDENTIFIER "defiguru-vpn" CACHE STRING "Local notifications")
set(CLIENT_ENABLE_APP_UPDATES 0 CACHE BOOL "Never install upstream-branded updates")
set(BUILD_OSX_APP_IDENTIFIER "ai.defiguru.vpn" CACHE STRING "macOS bundle")
set(BUILD_IOS_APP_IDENTIFIER "ai.defiguru.vpn" CACHE STRING "iOS bundle")
set(BUILD_IOS_GROUP_IDENTIFIER "group.ai.defiguru.vpn" CACHE STRING "Shared app group")
set(CLIENT_IOS_BUNDLE_ID "ai.defiguru.vpn" CACHE STRING "Store lookup bundle")
set(CLIENT_IOS_STORE_URL_FALLBACK "https://defiguru.ai/services/vpn" CACHE STRING "Release information")
set(CLIENT_ANDROID_PACKAGE "ai.defiguru.vpn" CACHE STRING "Android application ID")
set(CLIENT_EXTRA_QRC "${CMAKE_CURRENT_LIST_DIR}/defiguru.qrc" CACHE STRING "DeFiGuru brand resources")
# Signing belongs to the owner. Empty entries intentionally shadow upstream IDs.
set(BUILD_VPN_DEVELOPMENT_TEAM "" CACHE STRING "Owner Apple developer team")
foreach(PROFILE IN ITEMS
    CLIENT_IOS_PROVISIONING_PROFILE_SPECIFIER CLIENT_IOS_PROVISIONING_PROFILE_SPECIFIER_DEBUG
    CLIENT_IOS_NE_PROVISIONING_PROFILE_SPECIFIER CLIENT_IOS_NE_PROVISIONING_PROFILE_SPECIFIER_DEBUG
    CLIENT_MACOS_PROVISIONING_PROFILE_SPECIFIER CLIENT_MACOS_PROVISIONING_PROFILE_SPECIFIER_DEBUG
    CLIENT_MACOS_NE_PROVISIONING_PROFILE_SPECIFIER CLIENT_MACOS_NE_PROVISIONING_PROFILE_SPECIFIER_DEBUG)
    set(${PROFILE} "" CACHE STRING "Owner provisioning profile")
endforeach()
