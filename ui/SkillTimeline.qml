import QtQuick
import QtQuick.Controls
import QtQuick.Layouts
ColumnLayout {
    id: timeline
    property var player: ({})
    property real duration: 0
    property var events: player.timeline || []
    property var lanes: {
        const rows = [], seen = {}
        for (const event of events) {
            if (!seen[event.skill]) { seen[event.skill] = true; rows.push({skill: event.skill, name: event.name, icon: event.icon}) }
        }
        return rows
    }
    property real firstTime: events.length ? Math.min(0, Number(events[0].time)) : 0
    property real lastTime: Math.max(1, duration, events.length ? Number(events[events.length - 1].time) : 0)
    property real scale: zoom.value
    property string hoverText: ""
    spacing: 8
    RowLayout {
        Layout.fillWidth: true
        Label {text: "Observed skill casts · hover a marker for details"; color: "#9eadc3"; Layout.fillWidth: true}
        Label {text: "Zoom"; color: "#9eadc3"}
        Slider {id: zoom; from: 4; to: 80; value: 20; Layout.preferredWidth: 160}
    }
    Label {
        visible: timeline.events.length === 0
        text: "No skill usage timestamps recorded for this player. Older saved fights do not contain a timeline."
        color: "#9eadc3"; Layout.fillWidth: true; wrapMode: Text.WordWrap
    }
    Label {visible: timeline.player.timelineTruncated || false; text: "Timeline limited to the first 5,000 observed casts."; color: "#f1c778"}
    Flickable {
        id: view; objectName: "skillUsageTimeline"
        Layout.fillWidth: true; Layout.fillHeight: true; clip: true
        contentWidth: Math.max(width, 220 + (timeline.lastTime - timeline.firstTime) * timeline.scale)
        contentHeight: Math.max(height, 36 + timeline.lanes.length * 44)
        boundsBehavior: Flickable.StopAtBounds
        ScrollBar.horizontal: ScrollBar {}
        ScrollBar.vertical: ScrollBar {}
        onContentXChanged: plot.requestPaint()
        onContentYChanged: plot.requestPaint()
        Canvas {
            id: plot
            // Render only the visible viewport, even for long fights.
            x: view.contentX; y: view.contentY; width: view.width; height: view.height
            onWidthChanged: requestPaint()
            onHeightChanged: requestPaint()
            onImageLoaded: requestPaint()
            onPaint: {
                const ctx = getContext("2d")
                ctx.clearRect(0, 0, width, height)
                ctx.fillStyle = "#1b2433"; ctx.fillRect(0, 0, width, height)
                const ox = view.contentX, oy = view.contentY, labelWidth = 190
                const positions = {}
                ctx.font = "12px sans-serif"
                for (let i = 0; i < timeline.lanes.length; ++i) {
                    const lane = timeline.lanes[i], y = 36 + i * 44 - oy
                    positions[lane.skill] = i
                    if (y + 44 < 0 || y > height) continue
                    ctx.fillStyle = i % 2 ? "#202a3b" : "#1b2433"; ctx.fillRect(0, y, width, 44)
                }
                ctx.save();ctx.beginPath();ctx.rect(labelWidth, 0, Math.max(0,width-labelWidth), height);ctx.clip()
                const step = Math.max(1, Math.ceil(80 / timeline.scale))
                const start = Math.floor((timeline.firstTime + ox / timeline.scale) / step) * step
                for (let t = start; t <= timeline.lastTime; t += step) {
                    const x = labelWidth + (t - timeline.firstTime) * timeline.scale - ox
                    if (x > width) break
                    ctx.strokeStyle = "#354157";ctx.beginPath();ctx.moveTo(x,0);ctx.lineTo(x,height);ctx.stroke()
                    ctx.fillStyle = "#9eadc3";ctx.fillText(t.toFixed(0) + "s", x+4, 20)
                }
                for (const event of timeline.events) {
                    const x = labelWidth + (event.time - timeline.firstTime) * timeline.scale - ox
                    const y = 36 + positions[event.skill] * 44 + 22 - oy
                    if (x < labelWidth-13 || x > width+13 || y < -13 || y > height+13) continue
                    ctx.fillStyle = "#34435a";ctx.fillRect(x-13,y-13,26,26)
                    const url = event.icon ? "qrc:/skill-icons/" + event.icon : ""
                    if (url && isImageLoaded(url)) ctx.drawImage(url,x-12,y-12,24,24)
                    else {
                        if (url && !isImageError(url)) loadImage(url)
                        ctx.fillStyle = "#c0cbdc";ctx.fillText("?",x-4,y+4)
                    }
                }
                ctx.restore()
                ctx.fillStyle = "#263145";ctx.fillRect(0,0,labelWidth,height)
                ctx.fillStyle = "#e6edf6";ctx.fillText("Skill",8,20)
                for (let i = 0; i < timeline.lanes.length; ++i) {
                    const lane=timeline.lanes[i], y=36+i*44-oy
                    if(y+44<0 || y>height) continue
                    if(lane.icon) {
                        const url="qrc:/skill-icons/"+lane.icon
                        if(!isImageLoaded(url)) loadImage(url)
                        else ctx.drawImage(url,8,y+8,28,28)
                    }
                    ctx.save();ctx.beginPath();ctx.rect(42,y,labelWidth-48,44);ctx.clip()
                    ctx.fillStyle="#e6edf6";ctx.fillText(lane.name,42,y+26);ctx.restore()
                }
            }
            MouseArea {
                anchors.fill: parent; hoverEnabled: true; acceptedButtons: Qt.NoButton
                onExited: timeline.hoverText = ""
                onPositionChanged: mouse => {
                    timeline.hoverText = ""
                    if(mouse.x < 190) return
                    const laneIndex=Math.floor((mouse.y+view.contentY-36)/44)
                    if(laneIndex<0 || laneIndex>=timeline.lanes.length) return
                    const skill=timeline.lanes[laneIndex].skill
                    const time=timeline.firstTime+(mouse.x+view.contentX-190)/timeline.scale
                    let nearest=null, distance=Infinity
                    for(const event of timeline.events) if(event.skill===skill && Math.abs(event.time-time)<distance) {nearest=event;distance=Math.abs(event.time-time)}
                    if(nearest && distance*timeline.scale<=13) timeline.hoverText=nearest.name+" · "+Number(nearest.time).toFixed(2)+"s · Cast observed"
                }
                ToolTip.visible: timeline.hoverText.length > 0
                ToolTip.text: timeline.hoverText
            }
        }
        Connections {
            target: timeline
            function onEventsChanged() {plot.requestPaint()}
            function onScaleChanged() {plot.requestPaint()}
            function onDurationChanged() {plot.requestPaint()}
        }
    }
}
