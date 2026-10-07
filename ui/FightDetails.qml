import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
Window {
    id: details
    objectName: "fightDetailsWindow"
    visible: false
    width: 1240; height: 720; minimumWidth: 850; minimumHeight: 480
    color: "#19202d"
    flags: Qt.Window
    title: (backend.detailPlayer.name || "Player") + " · SlopMeter damage breakdown"
    property var player: backend.detailPlayer
    property var fight: backend.detailFight
    function number(value) {return Number(value || 0).toLocaleString(Qt.locale(), 'f', 0)}
    function percent(value) {return Number(value || 0).toFixed(1) + "%"}
    function selectedPlayerIndex() {
        for (let i=0;i<backend.displayPlayers.length;++i) if (backend.displayPlayers[i].id === player.id) return i
        return 0
    }
    component Cell: Label {
        color: "#e6edf6"; font.pixelSize: 12; horizontalAlignment: Text.AlignRight
        elide: Text.ElideRight; Layout.alignment: Qt.AlignVCenter
    }
    ColumnLayout {
        anchors.fill: parent; anchors.margins: 20; spacing: 14
        RowLayout {
            Layout.fillWidth: true
            Label {text: "Damage breakdown"; color: "#f2f5fa"; font.pixelSize: 22; font.bold: true; Layout.fillWidth: true}
            ComboBox {
                Layout.preferredWidth: 350
                model: backend.fightChoices; textRole: "label"; valueRole: "id"
                currentIndex: backend.selectedFightIndex
                onActivated: backend.selectFight(currentValue)
            }
            ComboBox {
                Layout.preferredWidth: 190
                model: backend.displayPlayers; textRole: "name"; valueRole: "id"
                currentIndex: details.selectedPlayerIndex()
                onActivated: backend.selectPlayer(currentValue)
            }
        }
        RowLayout {
            Layout.fillWidth: true; spacing: 12
            Image {source: backend.classIcon(details.player.class || ""); sourceSize.width: 48; sourceSize.height: 48; Layout.preferredWidth: 44; Layout.preferredHeight: 44}
            ColumnLayout {
                Layout.fillWidth: true; spacing: 3
                Label {text: (details.player.name || "No player selected") + (details.player.class ? " · " + details.player.class : ""); color: "#ffffff"; font.pixelSize: 18; font.bold: true}
                Label {text: ((details.fight.boss || {}).name || (details.fight.scene || {}).name || "Combat") + " · " + Number(details.fight.duration || 0).toFixed(1) + "s · " + (details.fight.active ? "Live" : details.fight.encounter || "Completed"); color: "#9eadc3"}
            }
            ColumnLayout {
                spacing: 3
                Label {text: details.number(details.player.damage) + " damage"; color: "#ffffff"; font.pixelSize: 18; font.bold: true; Layout.alignment: Qt.AlignRight}
                Label {text: details.number(details.player.dps) + " DPS · " + details.percent(details.player.share) + " of party damage"; color: "#9ad9ec"; Layout.alignment: Qt.AlignRight}
            }
        }
        Rectangle {Layout.fillWidth: true; implicitHeight: 1; color: "#42516a"}
        TabBar {
            id: analysisTabs; objectName: "analysisTabs"; Layout.fillWidth: true
            TabButton {text: "Damage breakdown"}
            TabButton {text: "Skill timeline"}
        }
        SkillTimeline {
            visible: analysisTabs.currentIndex === 1
            Layout.fillWidth: true; Layout.fillHeight: true
            player: details.player; duration: Number(details.fight.duration || 0)
        }
        ScrollView {
            visible: analysisTabs.currentIndex === 0
            id: table
            Layout.fillWidth: true; Layout.fillHeight: true; clip: true
            contentWidth: Math.max(availableWidth,1190)
            Column {
                width: table.contentWidth; spacing: 4
                Rectangle {
                    width: parent.width; height: 36; color: "#263145"
                    RowLayout {
                        anchors.fill: parent; anchors.leftMargin: 8; anchors.rightMargin: 8; spacing: 8
                        Cell {text: "Skill · highest damage first"; Layout.fillWidth: true; horizontalAlignment: Text.AlignLeft; font.bold: true}
                        Cell {text: "Damage"; Layout.preferredWidth: 110; font.bold: true}
                        Cell {text: "Player %"; Layout.preferredWidth: 68; font.bold: true}
                        Cell {text: "Party %"; Layout.preferredWidth: 68; font.bold: true}
                        Cell {text: "Crit %"; Layout.preferredWidth: 60; font.bold: true}
                        Cell {text: "Min"; Layout.preferredWidth: 80; font.bold: true}
                        Cell {text: "Max"; Layout.preferredWidth: 80; font.bold: true}
                        Cell {text: "Hits"; Layout.preferredWidth: 50; font.bold: true}
                        Cell {text: "Hits/s"; Layout.preferredWidth: 60; font.bold: true}
                        Cell {text: "Uses"; Layout.preferredWidth: 50; font.bold: true}
                    }
                }
                ListView {
                    id: skillList
                    objectName: "skillBreakdown"
                    width: parent.width; height: count * 52
                    interactive: false; model: skillModel
                    delegate: Rectangle {
                        id: skill
                        required property int index
                        required property int skillId
                        required property string skillName
                        required property string skillIcon
                        required property real damage
                        required property real share
                        required property real fightShare
                        required property real criticalRate
                        required property bool criticalKnown
                        required property real minDamage
                        required property real maxDamage
                        required property real hits
                        required property real hitsPerSecond
                        required property real uses
                        required property bool usesKnown
                        width: skillList.width; height: 52
                        color: index % 2 ? "#202a3b" : "#1b2433"
                        Rectangle {width: parent.width * skill.share / 100; height: parent.height; color: "#173f536b"}
                        RowLayout {
                            anchors.fill: parent; anchors.leftMargin: 8; anchors.rightMargin: 8; spacing: 8
                            RowLayout {
                                Layout.fillWidth: true; Layout.minimumWidth: 0; spacing: 8
                                Rectangle {
                                    Layout.preferredWidth: 36; Layout.preferredHeight: 36; color: "#34435a"; radius: 4
                                    Image {id: icon; anchors.fill: parent; source: skill.skillIcon ? "qrc:/skill-icons/" + skill.skillIcon : ""; sourceSize.width: 40; sourceSize.height: 40}
                                    Label {anchors.centerIn: parent; text: "?"; color: "#9eadc3"; visible: icon.status !== Image.Ready}
                                }
                                ColumnLayout {
                                    Layout.fillWidth: true; Layout.minimumWidth: 0; spacing: 1
                                    Label {text: skill.skillName; color: "#ffffff"; font.bold: true; Layout.fillWidth: true; elide: Text.ElideRight}
                                    Label {text: "ID " + skill.skillId; color: "#8e9eb5"; font.pixelSize: 10}
                                }
                            }
                            Cell {text: details.number(skill.damage); Layout.preferredWidth: 110}
                            Cell {text: details.percent(skill.share); Layout.preferredWidth: 68}
                            Cell {text: details.percent(skill.fightShare); Layout.preferredWidth: 68}
                            Cell {text: skill.criticalKnown ? details.percent(skill.criticalRate) : "—"; Layout.preferredWidth: 60}
                            Cell {text: skill.hits ? details.number(skill.minDamage) : "—"; Layout.preferredWidth: 80}
                            Cell {text: skill.hits ? details.number(skill.maxDamage) : "—"; Layout.preferredWidth: 80}
                            Cell {text: details.number(skill.hits); Layout.preferredWidth: 50}
                            Cell {text: skill.hitsPerSecond.toFixed(2); Layout.preferredWidth: 60}
                            Cell {text: skill.usesKnown ? details.number(skill.uses) : "—"; Layout.preferredWidth: 50}
                        }
                    }
                }
                Label {visible: skillList.count === 0; text: "No decoded skills for this player yet."; color: "#9eadc3"; padding: 16}
            }
        }
        Label {
            visible: analysisTabs.currentIndex === 0
            Layout.fillWidth: true; wrapMode: Text.WordWrap; color: "#9eadc3"; font.pixelSize: 12
            text: "Player % is this skill's share of the player's damage; Party % uses all participants. Uses counts observed casts, not individual hits. Missing cast data is shown as —."
        }
    }
}
