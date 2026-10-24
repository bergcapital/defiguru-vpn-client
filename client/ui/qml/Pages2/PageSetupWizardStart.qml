import QtQuick
import QtQuick.Controls
import QtQuick.Layouts

import PageEnum 1.0
import Style 1.0

import "./"
import "../Controls2"
import "../Config"
import "../Controls2/TextTypes"
import "../Components"

PageType {
    id: root
    enableTimer: false

    function previewConnection() {
        if (ImportController.extractConfigFromData(connectionKey.textField.text)) {
            PageController.goToPage(PageEnum.PageSetupWizardViewConfig)
        }
    }

    ScrollView {
        anchors.fill: parent
        contentWidth: availableWidth
        clip: true

    ColumnLayout {
        id: content

        width: root.width
        spacing: 0

        BasicButtonType {
            text: LanguageUiController.currentLanguageName + "  ⌄"
            defaultColor: AmneziaStyle.color.transparent
            hoveredColor: AmneziaStyle.color.translucentWhite
            pressedColor: AmneziaStyle.color.sheerWhite
            textColor: "#C7BCE8"
            Layout.alignment: Qt.AlignRight
            Layout.topMargin: 12 + PageController.safeAreaTopMargin
            Layout.rightMargin: 16
            clickedFunc: function() { languageDrawer.openTriggered() }
        }

        Image {
            id: image
            source: "qrc:/defiguru/brain.webp"

            Layout.alignment: Qt.AlignHCenter | Qt.AlignVCenter
            Layout.topMargin: 16
            Layout.preferredWidth: 180
            Layout.preferredHeight: 180
        }

        Header2TextType {
            text: "DeFiGuru VPN"
            horizontalAlignment: Text.AlignHCenter
            Layout.fillWidth: true
        }

        ParagraphTextType {
            text: qsTr("Paste your personal connection. Then tap Connect.")
            horizontalAlignment: Text.AlignHCenter
            Layout.fillWidth: true
            Layout.leftMargin: 24
            Layout.rightMargin: 24
        }

        TextFieldWithHeaderType {
            id: connectionKey
            headerText: qsTr("Your connection")
            buttonText: qsTr("Paste")
            buttonDefaultColor: "#763AC0"
            buttonHoveredColor: "#8950D2"
            buttonPressedColor: "#602D9F"
            buttonTextColor: "#FFFFFF"
            textField.placeholderText: "vpn://"
            Layout.fillWidth: true
            Layout.topMargin: 24
            Layout.leftMargin: 20
            Layout.rightMargin: 20
            clickedFunc: function() {
                connectionKey.textField.text = ""
                connectionKey.textField.paste()
                if (connectionKey.textField.text.trim() !== "") {
                    root.previewConnection()
                }
            }
        }

        BasicButtonType {
            id: startButton
            Layout.fillWidth: true
            Layout.bottomMargin: 8
            Layout.topMargin: 16
            Layout.leftMargin: 16
            Layout.rightMargin: 16
            Layout.alignment: Qt.AlignBottom

            text: qsTr("Add my VPN")
            visible: connectionKey.textField.text.trim() !== ""
            enabled: visible

            clickedFunc: function() {
                root.previewConnection()
            }
        }

        BasicButtonType {
            text: qsTr("Import a file or QR code")
            defaultColor: AmneziaStyle.color.transparent
            hoveredColor: AmneziaStyle.color.translucentWhite
            pressedColor: AmneziaStyle.color.sheerWhite
            textColor: "#C7BCE8"
            Layout.fillWidth: true
            Layout.leftMargin: 20
            Layout.rightMargin: 20
            clickedFunc: function() {
                PageController.goToPage(PageEnum.PageSetupWizardConfigSource)
            }
        }

        BasicButtonType {
            text: qsTr("Help & my account")
            defaultColor: AmneziaStyle.color.transparent
            hoveredColor: AmneziaStyle.color.translucentWhite
            pressedColor: AmneziaStyle.color.sheerWhite
            textColor: "#AEB7CD"
            Layout.fillWidth: true
            clickedFunc: function() {
                Qt.openUrlExternally("https://t.me/DeFiGuru_VPN_bot")
            }
        }

        CaptionTextType {
            text: "Powered by YLDX.AI"
            horizontalAlignment: Text.AlignHCenter
            Layout.fillWidth: true
            Layout.bottomMargin: 20 + PageController.safeAreaBottomMargin
        }
    }

    }

    SelectLanguageDrawer {
        id: languageDrawer
        width: root.width
        height: root.height
    }

    Timer {
        interval: 250
        running: SettingsController.isOnTv()
        repeat: true
        onTriggered: {
            connectionKey.textField.forceActiveFocus()
            if (connectionKey.textField.activeFocus) {
                running = false
            }
        }
    }

    onVisibleChanged: {
        if (visible && SettingsController.isOnTv()) {
            connectionKey.textField.forceActiveFocus()
        }
    }
}
