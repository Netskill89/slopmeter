import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
Window {
    id: settings
    objectName: "settingsWindow"
    visible: false
    width: 440; height: 740; minimumWidth: 360; minimumHeight: 400
    color: "#19202d"; flags: Qt.Window
    title: "SlopMeter · Settings"
    ScrollView {
        anchors.fill: parent; anchors.margins: 20
        clip: true; contentWidth: availableWidth
        ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
        ColumnLayout {
            width: parent.width; spacing: 6
            Label {text: "Meter settings"; color: "#f2f5fa"; font.pixelSize: 22; font.bold: true}
            Label {text: "Changes apply immediately and are saved automatically."; color: "#9eadc3"; Layout.fillWidth: true; wrapMode: Text.WordWrap}
            Label {text: "Layout"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            CheckBox {objectName: "testModeToggle"; palette.windowText: "#e6edf6"; text: "Test mode"; checked: backend.testMode; onToggled: backend.testMode = checked}
            CheckBox {objectName: "borderToggle"; palette.windowText: "#e6edf6"; text: "Show meter border"; checked: backend.showBorder; onToggled: backend.showBorder = checked}
            Label {text: "Player bar style"; color: "#c0cbdc"}
            ComboBox {objectName: "barStyleSelector"; Layout.fillWidth: true; model: ["Classic (default)", "Soft gradient", "Inset shadow", "Glossy"]; currentIndex: backend.barStyle; onActivated: backend.barStyle = currentIndex}
            Label {text: "Width · " + backend.barWidth + " px"; color: "#c0cbdc"}
            Slider {objectName: "barWidthSlider"; Layout.fillWidth: true; from: 240; to: 720; stepSize: 8; value: backend.barWidth; onMoved: backend.barWidth = Math.round(value)}
            Label {text: "Height · " + backend.barHeight + " px"; color: "#c0cbdc"}
            Slider {objectName: "barHeightSlider"; Layout.fillWidth: true; from: 28; to: 80; stepSize: 2; value: backend.barHeight; onMoved: backend.barHeight = Math.round(value)}
            Label {text: "Vertical spacing · " + backend.barSpacing + " px"; color: "#c0cbdc"}
            Slider {objectName: "barSpacingSlider"; Layout.fillWidth: true; from: 0; to: 20; stepSize: 1; value: backend.barSpacing; onMoved: backend.barSpacing = Math.round(value)}
            CheckBox {palette.windowText: "#e6edf6"; text: "Show damage and share"; checked: backend.showDetails; onToggled: backend.showDetails = checked}
            Label {text: "Opacity"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            Label {text: "Background opacity · " + backend.backgroundOpacity + "%"; color: "#c0cbdc"}
            Slider {objectName: "backgroundOpacitySlider"; Layout.fillWidth: true; from: 0; to: 100; stepSize: 1; value: backend.backgroundOpacity; onMoved: backend.backgroundOpacity = Math.round(value)}
            Label {text: "Affects the blue background only."; color: "#9eadc3"; font.pixelSize: 11}
            Label {text: "Overall meter opacity · " + backend.overallOpacity + "%"; color: "#c0cbdc"}
            Slider {objectName: "overallOpacitySlider"; Layout.fillWidth: true; from: 0; to: 100; stepSize: 1; value: backend.overallOpacity; onMoved: backend.overallOpacity = Math.round(value)}
            Label {text: "Fades the background, bars and text together. Settings stays visible; reopen it from the tray if the meter is invisible."; color: "#9eadc3"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {text: "Network capture"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            ComboBox {
                id: interfaceSelector; objectName: "captureInterfaceSelector"
                Layout.fillWidth: true; enabled: !backend.captureReplay
                model: backend.captureInterfaces; textRole: "label"; valueRole: "name"
                currentIndex: {
                    const rows = backend.captureInterfaces
                    const selected = backend.captureInterface
                    for (let i = 0; i < rows.length; ++i)
                        if (rows[i].name === selected) return i
                    return -1
                }
                onActivated: backend.captureInterface = currentValue
            }
            RowLayout {
                Button {text: "Refresh interfaces"; onClicked: backend.refreshInterfaces()}
                Button {text: "Apply and restart capture"; enabled: !backend.captureReplay; onClicked: backend.applyCaptureInterface()}
            }
            Label {text: backend.captureReplay ? "Interface selection is disabled during replay." : "Automatic uses the capture helper’s default interface. Choose your Ethernet, Wi-Fi, VPN, or all interfaces if needed. Applying ends the current fight and may request capture permission again."; color: "#9eadc3"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {text: backend.status; color: "#9eadc3"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {text: backend.interfaceStatus; visible: text.length > 0; color: "#c0cbdc"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {text: "Boss health: " + backend.bossHealthStatus; visible: backend.bossHealthStatus.length > 0; color: "#c0cbdc"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {text: "Data refresh"; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            Label {text: "Data polling interval · " + backend.pollInterval + " ms"; color: "#c0cbdc"}
            Slider {objectName: "pollIntervalSlider"; Layout.fillWidth: true; from: 50; to: 1000; stepSize: 50; snapMode: Slider.SnapAlways; value: backend.pollInterval; onMoved: backend.pollInterval = Math.round(value)}
            Label {text: "Packets are captured continuously. This controls how often the meter refreshes. Default: 200 ms."; color: "#9eadc3"; Layout.fillWidth: true; font.pixelSize: 11; wrapMode: Text.WordWrap}
            Label {objectName: "cpuWarning"; visible: backend.pollInterval < 200; text: "Warning: intervals below 200 ms can cause high CPU usage."; color: "#f1c778"; Layout.fillWidth: true; font.pixelSize: 12; wrapMode: Text.WordWrap}
            Label {text: "SlopMeter " + backend.appVersion; color: "#9ad9ec"; font.bold: true; Layout.topMargin: 10}
            RowLayout {
                Layout.fillWidth: true; Layout.topMargin: 10
                Button {text: "Restore defaults"; onClicked: backend.resetAppearance()}
                Item {Layout.fillWidth: true}
                Button {text: "Close"; onClicked: settings.close()}
            }
        }
    }
}
