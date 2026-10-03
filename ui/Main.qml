import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
Window {
    id: meter
    visible: false; width: 800; height: 600; color: "transparent"
    title: "SLOPMETER"
    flags: Qt.FramelessWindowHint | Qt.WindowDoesNotAcceptFocus
    function openSettings() {
        settingsLoader.active = true
        if (settingsLoader.item) {settingsLoader.item.screen = meter.screen;settingsLoader.item.show();settingsLoader.item.raise();settingsLoader.item.requestActivate()}
    }
    Loader {id: settingsLoader; active: false; source: "Settings.qml"}
    function openDetails(actorId) {
        backend.selectPlayer(actorId)
        detailLoader.active = true
        if (detailLoader.item) {detailLoader.item.screen = meter.screen;detailLoader.item.show();detailLoader.item.raise();detailLoader.item.requestActivate()}
    }
    Loader {id: detailLoader; active: false; source: "FightDetails.qml"}
    function clock(seconds) { const value = Math.max(0, Math.floor(Number(seconds || 0))); return Math.floor(value/60) + ":" + (value%60).toString().padStart(2,"0") }
    function number(value) { return Number(value || 0).toLocaleString(Qt.locale(), 'f', 0) }
    function classColor(name) {
        const colors = {Gladiator: "#ba7449", Templar: "#537ecd", Ranger: "#548d50", Assassin: "#9972c7",
            Elementalist: "#42a08f", Sorcerer: "#b45469", Cleric: "#b89b44", Chanter: "#c479af", Brawler: "#c05d3c"}
        return colors[name] || "#53677c"
    }
    Rectangle {
        id: panel
        objectName: "meterPanel"
        x: 40; y: 100
        width: Math.min(backend.barWidth + 28, meter.width)
        height: Math.min(content.implicitHeight + 28, meter.height)
        radius: 10; color: Qt.rgba(25/255,32/255,45/255,backend.backgroundOpacity/100); border.color: "#42516a"; border.width: backend.showBorder ? 1 : 0
        opacity: backend.overallOpacity / 100
        layer.enabled: opacity < 1
        ScrollView {
            anchors.fill: parent; anchors.margins: 14
            clip: true; contentWidth: availableWidth
            ScrollBar.horizontal.policy: ScrollBar.AlwaysOff
            ScrollBar.vertical.policy: ScrollBar.AsNeeded
            ColumnLayout {
                id: content
                width: parent.width; spacing: 8
                RowLayout {
                    Layout.fillWidth: true
                    Item {
                        Layout.fillWidth: true; implicitHeight: 32
                        RowLayout {
                            anchors.fill: parent; spacing: 8
                            Label {text: "⠿  SLOPMETER"; color: "#9ad9ec"; font.bold: true; font.pixelSize: 11}
                            Label {
                                objectName: "headerCharacter"
                                text: backend.character || "Relog"; color: backend.character ? "#f2f5fa" : "#f1c778"
                                Layout.fillWidth: true; Layout.minimumWidth: 0; elide: Text.ElideRight; horizontalAlignment: Text.AlignRight; font.pixelSize: 11
                                ToolTip.visible: characterHover.hovered; ToolTip.text: backend.character || "Character not detected. Relog with capture running."
                                HoverHandler {id: characterHover}
                            }
                        }
                        DragHandler {
                            target: panel
                            xAxis.minimum: 0; xAxis.maximum: Math.max(0,meter.width-panel.width)
                            yAxis.minimum: 0; yAxis.maximum: Math.max(0,meter.height-panel.height)
                            onActiveChanged: { if(active) backend.beginDrag(); else backend.endDrag() }
                            cursorShape: active ? Qt.ClosedHandCursor : Qt.OpenHandCursor
                        }
                    }
                    Button { text: "⚙"; implicitWidth: 32; implicitHeight: 32; onClicked: meter.openSettings(); ToolTip.visible: hovered; ToolTip.text: "Meter settings" }
                    Button { text: "−"; implicitWidth: 32; implicitHeight: 32; onClicked: backend.hide(); ToolTip.visible: hovered; ToolTip.text: "Hide; restore from the tray" }
                    Button { text: "×"; implicitWidth: 32; implicitHeight: 32; onClicked: backend.quit() }
                }
                ComboBox {
                    Layout.fillWidth: true
                    model: backend.fightChoices; textRole: "label"; valueRole: "id"
                    currentIndex: backend.selectedFightIndex
                    onActivated: backend.selectFight(currentValue)
                    visible: backend.fightChoices.length > 1
                    ToolTip.visible: hovered; ToolTip.text: "Last 10 fights · choose Current fight to return to live data"
                }
                ColumnLayout {
                    Layout.fillWidth: true; visible: !!backend.boss.entity; spacing: 3
                    Rectangle {
                        objectName: "targetHealthBar"
                        Layout.fillWidth: true; implicitHeight: 26; radius: 4; color: "#423139"
                        Rectangle {
                            width: parent.width * (backend.boss.maxKnown ? backend.boss.percent / 100 : 0)
                            height: parent.height; radius: 4; color: "#a44b57"
                            Behavior on width { NumberAnimation { duration: 120 } }
                        }
                        RowLayout {
                            anchors.fill: parent; anchors.leftMargin: 6; anchors.rightMargin: 6; spacing: 6
                            Label {text: backend.boss.name || "Target"; color: "#fff1eb"; font.pixelSize: 11; font.bold: true; Layout.fillWidth: true; Layout.minimumWidth: 0; elide: Text.ElideRight}
                            Label {
                                color: "#fff1eb"; font.pixelSize: 11; Layout.maximumWidth: parent.width * 0.65; elide: Text.ElideRight
                                text: !backend.boss.known ? "HP unknown" : backend.boss.maxKnown ? meter.number(backend.boss.hp) + " / " + meter.number(backend.boss.max) + " · " + Number(backend.boss.percent).toFixed(1) + "%" : meter.number(backend.boss.hp) + " HP"
                            }
                        }
                        HoverHandler {id: targetHover}
                        ToolTip.visible: targetHover.hovered
                        ToolTip.text: (backend.boss.name || "Target") + (!backend.boss.known ? " · Waiting for captured HP" : !backend.boss.maxKnown ? " · Maximum HP not detected" : "")
                    }
                }
                Label {
                    objectName: "combatStatus"
                    Layout.fillWidth: true; horizontalAlignment: Text.AlignRight; elide: Text.ElideRight; color: "#8e9eb5"; font.pixelSize: 11
                    text: backend.testMode ? "Test mode · " + meter.clock(backend.duration) : backend.detailFight.active ? "Combat · " + meter.clock(backend.duration) : backend.displayPlayers.length ? "Finished · " + meter.clock(backend.duration) : "Waiting for combat"
                    ToolTip.visible: combatHover.hovered; ToolTip.text: backend.encounter
                    HoverHandler {id: combatHover}
                }
                ListView {
                    id: bars; objectName: "playerBars"
                    Layout.fillWidth: true
                    implicitHeight: count ? count * backend.barHeight + Math.max(0,count-1) * backend.barSpacing : 36
                    Layout.preferredHeight: implicitHeight
                    interactive: false; spacing: backend.barSpacing
                    model: actorModel
                    delegate: Rectangle {
                        id: bar
                        property color classTint: meter.classColor(actorClass)
                        required property int actorId
                        required property string actorName
                        required property string actorClass
                        required property real damage
                        required property real dps
                        required property real share
                        required property real fill
                        TapHandler {onTapped: meter.openDetails(bar.actorId)}
                        HoverHandler {cursorShape: Qt.PointingHandCursor}
                        width: ListView.view.width; height: backend.barHeight; radius: 4; color: "#273348"
                        Rectangle {
                            id: damageFill
                            width: parent.width * parent.fill / 100; height: parent.height; radius: 4; color: bar.classTint
                            gradient: backend.barStyle === 1 ? softGradient : backend.barStyle === 3 ? glossyGradient : null
                            Gradient {id: softGradient; GradientStop {position: 0; color: Qt.lighter(bar.classTint,1.18)} GradientStop {position: 1; color: Qt.darker(bar.classTint,1.18)}}
                            Gradient {id: glossyGradient; GradientStop {position: 0; color: Qt.lighter(bar.classTint,1.32)} GradientStop {position: 0.48; color: Qt.lighter(bar.classTint,1.06)} GradientStop {position: 0.50; color: bar.classTint} GradientStop {position: 1; color: Qt.darker(bar.classTint,1.22)}}
                            Rectangle {visible: backend.barStyle === 2; anchors.left: parent.left; anchors.right: parent.right; anchors.top: parent.top; height: 8; radius: 4; gradient: Gradient {GradientStop {position: 0; color: "#65000000"} GradientStop {position: 1; color: "transparent"}}}
                            Rectangle {visible: backend.barStyle === 2; anchors.fill: parent; radius: 4; color: "transparent"; border.width: 1; border.color: "#45000000"}
                            Behavior on width { NumberAnimation { duration: 120 } }
                        }
                        RowLayout {
                            anchors.fill: parent; anchors.leftMargin: 6; anchors.rightMargin: 8; spacing: 7
                            Image {
                                source: backend.classIcon(bar.actorClass)
                                sourceSize.width: 32; sourceSize.height: 32
                                Layout.preferredWidth: Math.min(26,backend.barHeight-6); Layout.preferredHeight: Layout.preferredWidth
                                ToolTip.visible: iconHover.hovered; ToolTip.text: bar.actorClass || "Class not detected"
                                HoverHandler { id: iconHover }
                            }
                            Label { text: bar.actorName; color: "#ffffff"; font.bold: true; elide: Text.ElideRight; Layout.fillWidth: true; Layout.minimumWidth: 0 }
                            Label {
                                objectName: "compactShare"
                                visible: backend.showDetails && backend.barHeight < 38
                                text: bar.share.toFixed(1) + "%"; color: "#ffffff"; font.pixelSize: 11
                                Layout.preferredWidth: Math.max(40,bar.width * 0.15); horizontalAlignment: Text.AlignHCenter
                            }
                            ColumnLayout {
                                Layout.preferredWidth: backend.barHeight < 38 ? Math.max(80,bar.width / 2 - Math.max(40,bar.width * 0.15) / 2 - 8) : implicitWidth
                                spacing: 0
                                Label { text: meter.number(bar.dps) + " DPS"; color: "#ffffff"; font.bold: true; Layout.alignment: Qt.AlignRight }
                                Label { visible: backend.showDetails && backend.barHeight >= 38; text: meter.number(bar.damage) + " dmg · " + bar.share.toFixed(1) + "%"; color: "#e6edf6"; font.pixelSize: 11; Layout.alignment: Qt.AlignRight }
                            }
                        }
                    }
                    Label { anchors.centerIn: parent; visible: bars.count === 0; text: "Waiting for your party's damage…"; color: "#8e9eb5"; font.pixelSize: 12 }
                }
                RowLayout {
                    objectName: "meterFooter"
                    Layout.fillWidth: true; spacing: 8
                    Label {
                        objectName: "displayLabel"
                        text: backend.displayLabel; color: "#8e9eb5"; font.pixelSize: 11; elide: Text.ElideRight
                        Layout.fillWidth: true; Layout.minimumWidth: 0
                        ToolTip.visible: displayHover.hovered; ToolTip.text: backend.displayStatus
                        HoverHandler {id: displayHover}
                    }
                    Label {
                        objectName: "appVersionLabel"
                        text: "v" + backend.appVersion; color: "#8e9eb5"; font.pixelSize: 10
                    }
                    Rectangle {
                        objectName: "captureIndicator"
                        Layout.preferredWidth: 9; Layout.preferredHeight: 9; radius: 4.5
                        color: backend.captureActive ? "#54cf82" : "#ee6464"
                        Accessible.name: backend.captureActive ? "Capture active" : "Capture inactive"
                        HoverHandler {id: captureHover}
                        ToolTip.visible: captureHover.hovered; ToolTip.text: (backend.captureActive ? "Capture active" : "Capture inactive") + "\n" + backend.status
                    }
                }
            }
        }
    }
    onClosing: backend.quit()
}
