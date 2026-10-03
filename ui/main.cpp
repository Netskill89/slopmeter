#include <QApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QQmlProperty>
#include <QQuickWindow>
#include <QProcess>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QTimer>
#include <QSettings>
#include <QScreen>
#include <QAbstractListModel>
#include <QElapsedTimer>
#include <QSystemTrayIcon>
#include <QMenu>
#include <QPainter>
#include <LayerShellQt/Window>
#include <wayland-client.h>
#include <cstdio>
#include <cstring>
#include <QFile>
#include <QFileInfo>
#include <QDir>
#include <QTemporaryFile>
#include <QTemporaryDir>
#include <memory>
#include <cmath>
#include <QDBusConnection>
#include <QDBusInterface>
#include <QDBusPendingCallWatcher>
#include <QDBusPendingReply>
#include <QQuickItem>
#include <QQuickItemGrabResult>
#include <QMessageBox>
#include <QRegion>
#include <QSet>
#include <QEventLoop>
#include <QDateTime>
#include <QStandardPaths>
#include <QDesktopServices>
#include <QUrl>

class ActorModel : public QAbstractListModel {
public:
    QList<QVariantMap> rows;
    enum { Name=Qt::UserRole+1, Damage, DPS, Share, Fill, Class, ActorId };
    using QAbstractListModel::QAbstractListModel;
    int rowCount(const QModelIndex &parent={}) const override { return parent.isValid()?0:rows.size(); }
    QHash<int,QByteArray> roleNames() const override {static const QHash<int,QByteArray> roles{{Name,"actorName"},{Damage,"damage"},{DPS,"dps"},{Share,"share"},{Fill,"fill"},{Class,"actorClass"},{ActorId,"actorId"}};return roles;}
    QVariant data(const QModelIndex &i,int role) const override {
        if (!i.isValid()||i.row()>=rows.size()) return {};
        return rows[i.row()].value(QString::fromUtf8(roleNames().value(role)) == "actorName" ? "name" : role==Class ? "class" : role==ActorId ? "id" : QString::fromUtf8(roleNames().value(role)));
    }
    void update(const QJsonArray &array) {
        // Remove departing rows before insertion to avoid transient panel growth
        // when switching between real and simulated parties. Keep surviving delegates.
        QSet<qulonglong> ids;for(const auto &value:array) ids.insert(value.toObject().value("id").toVariant().toULongLong());
        for(int i=rows.size()-1;i>=0;--i) if(!ids.contains(rows[i].value("id").toULongLong())) {beginRemoveRows({},i,i);rows.removeAt(i);endRemoveRows();}
        for(int i=0;i<array.size();++i) {
            auto row=array[i].toObject().toVariantMap(); int existing=-1;
            for(int j=i;j<rows.size();++j) if(rows[j].value("id")==row.value("id")) { existing=j; break; }
            if(existing<0) { beginInsertRows({},i,i);rows.insert(i,row);endInsertRows(); }
            else {
                if(existing!=i) { beginMoveRows({},existing,existing,{},i);rows.move(existing,i);endMoveRows(); }
                if(rows[i]!=row) { rows[i]=row;emit dataChanged(index(i),index(i)); }
            }
        }
        if(rows.size()>array.size()) { beginRemoveRows({},array.size(),rows.size()-1);while(rows.size()>array.size()) rows.removeLast();endRemoveRows(); }
    }
};
class SkillModel : public ActorModel {
public:
    using ActorModel::ActorModel;
    QHash<int,QByteArray> roleNames() const override {
        static const QHash<int,QByteArray> roles=[] {
            QHash<int,QByteArray> roles;int key=Qt::UserRole+101;
            for(const auto &name:QList<QByteArray>{"skillId","skillName","skillIcon","damage","share","fightShare","hits","uses","usesKnown","critical","criticalRate","criticalKnown","minDamage","maxDamage","average","hitsPerSecond"}) roles.insert(key++,name);
            return roles;
        }();return roles;
    }
    QVariant data(const QModelIndex &i,int role) const override {
        if(!i.isValid()||i.row()>=rows.size()) return {};
        auto key=QString::fromUtf8(roleNames().value(role));
        if(key=="skillId") key="id";else if(key=="skillName") key="name";else if(key=="skillIcon") key="icon";else if(key=="minDamage") key="min";else if(key=="maxDamage") key="max";
        return rows[i.row()].value(key);
    }
};
class Bridge : public QObject {
    Q_OBJECT
    Q_PROPERTY(QVariantList fightChoices READ fightChoices NOTIFY historyChanged)
    Q_PROPERTY(int selectedFightIndex READ selectedFightIndex NOTIFY historyChanged)
    Q_PROPERTY(QVariantList displayPlayers READ displayPlayers NOTIFY detailChanged)
    Q_PROPERTY(QVariantMap detailPlayer READ detailPlayer NOTIFY detailChanged)
    Q_PROPERTY(QVariantMap detailFight READ detailFight NOTIFY detailChanged)
    Q_PROPERTY(bool testMode READ testMode WRITE setTestMode NOTIFY testModeChanged)
    Q_PROPERTY(QString appVersion READ appVersion CONSTANT)
    Q_PROPERTY(QString updateRepository READ updateRepository WRITE setUpdateRepository NOTIFY updateChanged)
    Q_PROPERTY(QString updateStatus READ updateStatus NOTIFY updateChanged)
    Q_PROPERTY(bool updateBusy READ updateBusy NOTIFY updateChanged)
    Q_PROPERTY(bool updateAvailable READ updateAvailable NOTIFY updateChanged)
    Q_PROPERTY(QString updatePath READ updatePath NOTIFY updateChanged)
    Q_PROPERTY(int barStyle READ barStyle WRITE setBarStyle NOTIFY appearanceChanged)
    Q_PROPERTY(bool showBorder READ showBorder WRITE setShowBorder NOTIFY appearanceChanged)
    Q_PROPERTY(QString captureInterface READ captureInterface WRITE setCaptureInterface NOTIFY interfacesChanged)
    Q_PROPERTY(QVariantList captureInterfaces READ captureInterfaces NOTIFY interfacesChanged)
    Q_PROPERTY(bool captureReplay READ captureReplay NOTIFY interfacesChanged)
    Q_PROPERTY(QString interfaceStatus READ interfaceStatus NOTIFY interfacesChanged)
    Q_PROPERTY(int pollInterval READ pollInterval WRITE setPollInterval NOTIFY pollingChanged)
    Q_PROPERTY(int barWidth READ barWidth WRITE setBarWidth NOTIFY appearanceChanged)
    Q_PROPERTY(int barHeight READ barHeight WRITE setBarHeight NOTIFY appearanceChanged)
    Q_PROPERTY(int barSpacing READ barSpacing WRITE setBarSpacing NOTIFY appearanceChanged)
    Q_PROPERTY(int backgroundOpacity READ backgroundOpacity WRITE setBackgroundOpacity NOTIFY appearanceChanged)
    Q_PROPERTY(int overallOpacity READ overallOpacity WRITE setOverallOpacity NOTIFY appearanceChanged)
    Q_PROPERTY(bool showDetails READ showDetails WRITE setShowDetails NOTIFY appearanceChanged)
    Q_PROPERTY(bool captureActive READ captureActive NOTIFY captureChanged)
    Q_PROPERTY(QString displayLabel READ displayLabel NOTIFY displayChanged)
    Q_PROPERTY(QString character READ character NOTIFY characterChanged)
    Q_PROPERTY(QString status READ status NOTIFY statusChanged)
    Q_PROPERTY(double duration READ duration NOTIFY durationChanged)
    Q_PROPERTY(QString encounter READ encounter NOTIFY encounterChanged)
    Q_PROPERTY(QVariantMap boss READ boss NOTIFY bossChanged)
    Q_PROPERTY(QString displayStatus READ displayStatus NOTIFY displayChanged)
    Q_CLASSINFO("D-Bus Interface", "org.aiondps.Screen")
public:
    bool previewEnabled=false;
    bool borderSetting=QSettings().value("appearance/showBorder",true).toBool();
    int pollSetting=QSettings().value("capture/pollIntervalMs",200).toInt();
    QElapsedTimer previewClock;
    QVariantList previewParty;
    QList<double> previewDamage;
    double previewLast=0;
    quint32 actorBeforePreview=0;
    static int normalizePoll(int value) {return ((qBound(50,value,1000)+25)/50)*50;}
    bool testMode() const {return previewEnabled;}
    int styleSetting=QSettings().value("appearance/barStyle",0).toInt();
    int barStyle() const {return qBound(0,styleSetting,3);}
    void setBarStyle(int value) {styleSetting=qBound(0,value,3);QSettings().setValue("appearance/barStyle",styleSetting);emit appearanceChanged();}
    bool showBorder() const {return borderSetting;}
    int pollInterval() const {return normalizePoll(pollSetting);}
    void setShowBorder(bool value) {borderSetting=value;QSettings().setValue("appearance/showBorder",value);emit appearanceChanged();}
    void sendRefreshInterval() {
        if(process.state()!=QProcess::NotRunning) process.write(QJsonDocument(QJsonObject{{"intervalMs",pollInterval()}}).toJson(QJsonDocument::Compact)+"\n");
    }
    void setPollInterval(int value) {
        value=normalizePoll(value);if(value==pollInterval()) return;
        pollSetting=value;QSettings().setValue("capture/pollIntervalMs",value);
        publish.setInterval(value);sendRefreshInterval();emit pollingChanged();
    }
    void setTestMode(bool enabled) {
        if(previewEnabled==enabled) return;
        previewEnabled=enabled;
        if(enabled) {
            actorBeforePreview=selectedActor;selectedActor=0;previewClock.start();previewLast=0;previewDamage.clear();
            for(const auto &value:previewParty) previewDamage.append(value.toMap().value("baseDps").toDouble()*60);
        } else selectedActor=actorBeforePreview;
        if(enabled) publish.start();else publish.stop();
        refreshDisplay();emit testModeChanged();emit historyChanged();emit statusChanged();
    }
    QVariantMap previewState() {
        const double elapsed=previewClock.elapsed()/1000.0,seconds=60+elapsed,delta=qMax(0.0,elapsed-previewLast);
        double largest=0,totalDamage=0;
        for(int i=0;i<previewParty.size();++i) {
            previewDamage[i]+=delta*previewParty[i].toMap().value("baseDps").toDouble()*(1+0.08*std::sin(elapsed*0.5+i));
            largest=qMax(largest,previewDamage[i]);totalDamage+=previewDamage[i];
        }
        previewLast=elapsed;QVariantList rows;
        for(int i=0;i<previewParty.size();++i) {
            auto row=previewParty[i].toMap();const double damage=previewDamage[i];
            row["damage"]=qRound64(damage);row["dps"]=damage/seconds;row["fill"]=damage/largest*100;row["share"]=damage/totalDamage*100;
            QVariantList skills;
            for(const auto &value:row.value("skills").toList()) {
                auto skill=value.toMap();const double skillDamage=damage*skill.value("weight").toDouble();
                const auto uses=qMax(1,qRound(seconds/3*skill.value("weight").toDouble()));const int hits=uses*2;
                skill["damage"]=qRound64(skillDamage);skill["share"]=skillDamage/damage*100;skill["fightShare"]=skillDamage/totalDamage*100;
                skill["hits"]=hits;skill["uses"]=uses;skill["usesKnown"]=true;skill["critical"]=qRound(hits*0.3);skill["criticalRate"]=30.0;skill["criticalKnown"]=true;
                skill["min"]=qRound(skillDamage/hits*0.75);skill["max"]=qRound(skillDamage/hits*1.25);skill["average"]=skillDamage/hits;skill["hitsPerSecond"]=hits/seconds;
                skills.append(skill);
            }
            row["skills"]=skills;rows.append(row);
        }
        const qlonglong maximumHP=8000000;
        const qlonglong currentHP=qRound64(maximumHP*qMax(0.05,0.68-elapsed*0.002));
        QVariantMap previewBoss{{"entity",900100},{"name","Fediv Wraith"},{"hp",currentHP},{"max",maximumHP},
            {"percent",100.0*currentHP/maximumHP},{"known",true},{"maxKnown",true}};
        return {{"character","Test party"},{"encounter","Test mode"},{"duration",seconds},{"active",true},{"session",0},{"actors",rows},{"boss",previewBoss}};
    }
    int widthSetting=QSettings().value("appearance/barWidth",352).toInt();
    int heightSetting=QSettings().value("appearance/barHeight",44).toInt();
    int spacingSetting=QSettings().value("appearance/barSpacing",4).toInt();
    int backgroundSetting=QSettings().value("appearance/backgroundOpacity",75).toInt();
    int overallSetting=QSettings().value("appearance/overallOpacity",100).toInt();
    bool detailsSetting=QSettings().value("appearance/showDetails",true).toBool();
    int barWidth() const {return qBound(240,widthSetting,720);}
    int barHeight() const {return qBound(28,heightSetting,80);}
    int barSpacing() const {return qBound(0,spacingSetting,20);}
    int backgroundOpacity() const {return qBound(0,backgroundSetting,100);}
    int overallOpacity() const {return qBound(0,overallSetting,100);}
    void setBackgroundOpacity(int value) {backgroundSetting=qBound(0,value,100);QSettings().setValue("appearance/backgroundOpacity",backgroundSetting);emit appearanceChanged();}
    void setOverallOpacity(int value) {overallSetting=qBound(0,value,100);QSettings().setValue("appearance/overallOpacity",overallSetting);emit appearanceChanged();}
    bool showDetails() const {return detailsSetting;}
    void setBarWidth(int value) {widthSetting=qBound(240,value,720);QSettings().setValue("appearance/barWidth",widthSetting);emit appearanceChanged();}
    void setBarHeight(int value) {heightSetting=qBound(28,value,80);QSettings().setValue("appearance/barHeight",heightSetting);emit appearanceChanged();}
    void setBarSpacing(int value) {spacingSetting=qBound(0,value,20);QSettings().setValue("appearance/barSpacing",spacingSetting);emit appearanceChanged();}
    void setShowDetails(bool value) {detailsSetting=value;QSettings().setValue("appearance/showDetails",value);emit appearanceChanged();}
    Q_INVOKABLE void resetAppearance() {setBarWidth(352);setBarHeight(44);setBarSpacing(4);setShowDetails(true);setBackgroundOpacity(75);setOverallOpacity(100);setShowBorder(true);setBarStyle(0);setPollInterval(200);}
    ActorModel actors;
    SkillModel skills;
    QProcess process;
    QByteArray buffer;
    QJsonObject pending;
    QTimer publish;
    QQuickWindow *window=nullptr;
    bool headless=false;
    QPoint dragOrigin;
    bool dragging=false, autoDisplay=true, detectionTest=false;
    QQuickItem *panel=nullptr;
    QString encounterLabel="Waiting for combat";
    QVariantMap bossState;
    bool captureReady=false,replayMode=false;
    QString displayMessage="Waiting for AION 2 window…", gameScreen, scriptName;
    QTemporaryFile script;
    Q_INVOKABLE QString displayStatus() const {return displayMessage;}
    QString encounter() const {return encounterLabel;}
    QVariantMap boss() const {return bossState;}
    bool captureActive() const {return captureReady;}
    void setCaptureReady(bool ready) {if(captureReady!=ready) {
        captureReady=ready;emit captureChanged();
        if(ready) {interfaceMessage="Capturing on "+(activeInterface.isEmpty()?"Automatic":activeInterface)+". Relog if no character is detected.";emit interfacesChanged();}
    }}
    QString displayLabel() const {auto name=window&&window->screen()?window->screen()->name():QString();return "Display - "+(name.isEmpty()?"Default":name);}
    void updateInputMask() {
        if(window&&panel) {
            // An empty QWindow mask resets the region. Use an off-surface region
            // so an entirely invisible meter lets all game clicks through.
            window->setMask(panel->opacity()>0?QRegion(QRectF(panel->x(),panel->y(),panel->width(),panel->height()).toAlignedRect()):QRegion(QRect(-1,-1,1,1)));
        }
    }
    void resizeCanvas() {if(window) {window->resize(window->screen()->geometry().size());updateInputMask();}}
    void selectScreen(QScreen *screen) {
        if(!screen||!window||dragging) return;
        if(window->screen()!=screen) {
            const bool visible=window->isVisible();const auto pos=position();
            window->hide();window->destroy();window->setScreen(screen);
#ifdef SLOPMETER_LAYER_SCREEN
            if(!headless) LayerShellQt::Window::get(window)->setScreen(screen);
#endif
            resizeCanvas();positionOverlay(pos);if(visible) window->show();
        }
        displayMessage=QString("%1 · %2").arg(autoDisplay?"Game display":"Manual display",screen->name());emit displayChanged();
    }
    void startGameDetection() {
        auto bus=QDBusConnection::sessionBus();bus.registerObject("/AionDPS",this,QDBusConnection::ExportAllSlots);
        QFile source(":/game-screen.js");if(!source.open(QIODevice::ReadOnly)||!script.open()) return;
        auto code=source.readAll();code.replace("@SERVICE@",bus.baseService().toUtf8());script.write(code);script.flush();
        scriptName=QString("aiondps-%1").arg(QCoreApplication::applicationPid());
        QDBusInterface scripting("org.kde.KWin","/Scripting","org.kde.kwin.Scripting",bus);
        auto watcher=new QDBusPendingCallWatcher(scripting.asyncCall("loadScript",script.fileName(),scriptName),this);
        connect(watcher,&QDBusPendingCallWatcher::finished,this,[this](QDBusPendingCallWatcher *call) {
            QDBusPendingReply<int> reply=*call;call->deleteLater();
            if(reply.isError()||reply.value()<0) {displayMessage="Automatic detection unavailable; choose a display in the tray.";emit displayChanged();return;}
            QDBusInterface running("org.kde.KWin",QString("/Scripting/Script%1").arg(reply.value()),"org.kde.kwin.Script",QDBusConnection::sessionBus());
            auto run=new QDBusPendingCallWatcher(running.asyncCall("run"),this);
            connect(run,&QDBusPendingCallWatcher::finished,this,[this](QDBusPendingCallWatcher *call) {
                if(call->isError()) {displayMessage="Game detection failed; choose a display in the tray.";emit displayChanged();}call->deleteLater();
            });
        });
    }
public slots:
    void gameOutput(const QString &output,const QString &title) {
        gameScreen=output;
        if(detectionTest) {
            fprintf(stderr,"KWin detection report: game output=%s title=%s\n",qPrintable(output),qPrintable(title));
            for(auto screen:QGuiApplication::screens()) fprintf(stderr,"Qt output: %s\n",qPrintable(screen->name()));
            QCoreApplication::exit(output.isEmpty()?6:0);return;
        }
        if(!autoDisplay) return;
        if(output.isEmpty()) {displayMessage="AION 2 window not found · select display in tray";emit displayChanged();return;}
        for(auto screen:QGuiApplication::screens()) if(screen->name()==output) {selectScreen(screen);return;}
    }
public:

    QVariantMap currentState,displayState;
    QVariantList history;
    quint64 selectedSession=0;
    quint32 selectedActor=0;
    QVariantList fightChoices() const {
        if(previewEnabled) return {QVariantMap{{"id",qulonglong(0)},{"label","Test mode"}}};
        QVariantList choices{QVariantMap{{"id",qulonglong(0)},{"label","Current fight"}}};
        for(const auto &value:history) {
            auto entry=value.toMap();auto state=entry.value("snapshot").toMap();auto boss=state.value("boss").toMap();
            auto label=boss.value("name").toString();if(label.isEmpty()) label="Open world";
            auto started=QDateTime::fromString(entry.value("started").toString(),Qt::ISODateWithMs).toLocalTime();
            choices.append(QVariantMap{{"id",state.value("session")},{"label",QString("#%1 · %2 · %3s · %4").arg(state.value("session").toULongLong()).arg(label).arg(state.value("duration").toDouble(),0,'f',1).arg(started.toString("MMM d HH:mm:ss"))}});
        }
        return choices;
    }
    int selectedFightIndex() const {
        if(previewEnabled) return 0;
        auto choices=fightChoices();for(int i=0;i<choices.size();++i) if(choices[i].toMap().value("id").toULongLong()==selectedSession) return i;return 0;
    }
    QVariantList displayPlayers() const {return displayState.value("actors").toList();}
    QVariantMap detailFight() const {return displayState;}
    QVariantMap detailPlayer() const {
        auto rows=displayPlayers();for(const auto &row:rows) if(row.toMap().value("id").toUInt()==selectedActor) return row.toMap();
        return rows.isEmpty()?QVariantMap{}:rows.first().toMap();
    }
    Q_INVOKABLE void selectPlayer(quint32 id) {selectedActor=id;updateDetail();}
    Q_INVOKABLE void selectFight(quint64 id) {if(previewEnabled) return;selectedSession=id;refreshDisplay();emit historyChanged();}
    Q_INVOKABLE QString classIcon(const QString &name) const {
        const QStringList supplied{"Gladiator","Templar","Ranger","Assassin","Sorcerer","Cleric","Chanter","Elementalist","Brawler"};
        return supplied.contains(name)?"qrc:/class-icons/"+name+".png":"qrc:/icons/Unknown.svg";
    }
    void refreshDisplay() {
        displayState=previewEnabled?previewState():currentState;
        if(selectedSession&&!previewEnabled) {
            bool found=false;
            for(const auto &value:history) {auto entry=value.toMap();auto state=entry.value("snapshot").toMap();if(state.value("session").toULongLong()==selectedSession) {displayState=state;displayState["started"]=entry.value("started");displayState["ended"]=entry.value("ended");found=true;break;}}
            if(!found) {selectedSession=0;emit historyChanged();}
        }
        // Keep the last completed fight on Current fight until positive damage
        // starts a new session. History owns immutable names, skills and target HP.
        if(!previewEnabled&&!selectedSession&&!currentState.value("active").toBool()&&currentState.value("actors").toList().isEmpty()) {
            for(const auto &value:history) {auto state=value.toMap().value("snapshot").toMap();if(state.value("session")==currentState.value("session")) {displayState=state;break;}}
        }
        auto next=(!previewEnabled&&!selectedSession?currentState:displayState).value("character").toString();if(next!=name) {name=next;emit characterChanged();}
        double seconds=displayState.value("duration").toDouble();if(seconds!=span) {span=seconds;emit durationChanged();}
        auto label=displayState.value("encounter","Waiting for combat").toString();if(label!=encounterLabel) {encounterLabel=label;emit encounterChanged();}
        auto boss=displayState.value("boss").toMap();if(boss!=bossState) {bossState=boss;emit bossChanged();}
        actors.update(QJsonArray::fromVariantList(displayPlayers()));updateDetail();
    }
    void updateDetail() {skills.update(QJsonArray::fromVariantList(detailPlayer().value("skills").toList()));emit detailChanged();}
    QString appVersion() const {return SLOPMETER_VERSION;}
    QString repositorySetting=QSettings().value("updates/repository",SLOPMETER_RELEASE_REPOSITORY).toString();
    QString updateMessage="Updates are checked only when requested.", downloadedPath;
    bool availableUpdate=false;
    QProcess updater;
    QString updateRepository() const {return repositorySetting;}
    QString updateStatus() const {return updateMessage;}
    bool updateBusy() const {return updater.state()!=QProcess::NotRunning;}
    bool updateAvailable() const {return availableUpdate;}
    QString updatePath() const {return downloadedPath;}
    void setUpdateRepository(const QString &value) {if(updateBusy()) return;repositorySetting=value.trimmed();QSettings().setValue("updates/repository",repositorySetting);availableUpdate=false;downloadedPath.clear();emit updateChanged();}
    Q_INVOKABLE void checkUpdates(bool download=false) {
        if(updateBusy()) return;
        updateMessage=download?"Downloading and verifying update…":"Checking latest release…";
        updater.setProgram(backendPath);
        QStringList arguments{download?"-download-update":"-check-update","-repository",repositorySetting,"-format",qEnvironmentVariableIsSet("APPIMAGE")?"AppImage":"tar.gz"};
        if(download) {auto directory=QStandardPaths::writableLocation(QStandardPaths::DownloadLocation);if(directory.isEmpty()) directory=QDir::homePath()+"/Downloads";arguments<<"-download-dir"<<directory;}
        updater.setArguments(arguments);updater.start();emit updateChanged();
    }
    Q_INVOKABLE void openUpdateFolder() {if(!downloadedPath.isEmpty()) QDesktopServices::openUrl(QUrl::fromLocalFile(QFileInfo(downloadedPath).absolutePath()));}
    QProcess interfaceLister;
    QString interfaceSetting=QSettings().value("capture/interface", "").toString();
    QVariantList interfaceRows{{QVariantMap{{"name",""},{"label","Automatic (capture default)"}}}};
    QString interfaceMessage;
    QStringList captureArguments;
    bool restartingCapture=false;
    QString activeInterface;
    QString captureInterface() const {return interfaceSetting;}
    QVariantList captureInterfaces() const {return interfaceRows;}
    bool captureReplay() const {return replayMode;}
    QString interfaceStatus() const {return interfaceMessage;}
    void setCaptureInterface(const QString &value) {
        if(replayMode||value==interfaceSetting) return;
        interfaceSetting=value;QSettings().setValue("capture/interface",value);
        interfaceMessage="Selection saved. Apply to restart capture.";emit interfacesChanged();
    }
    Q_INVOKABLE void refreshInterfaces() {
        if(interfaceLister.state()!=QProcess::NotRunning||backendPath.isEmpty()) return;
        interfaceLister.start(backendPath,{"-interfaces","-json"});
    }
    void startCapture() {
        auto arguments=captureArguments;
        activeInterface=interfaceSetting;
        if(!replayMode&&!activeInterface.isEmpty()) arguments<<"-interface"<<activeInterface;
        buffer.clear();pending={};setStatus(replayMode?"Starting replay…":"Starting capture…");process.start(backendPath,arguments);
    }
    Q_INVOKABLE void applyCaptureInterface() {
        if(replayMode||restartingCapture) return;
        if(process.state()!=QProcess::NotRunning&&interfaceSetting==activeInterface) {
            interfaceMessage="This interface is already active. Relog if no character is detected.";emit interfacesChanged();return;
        }
        interfaceMessage="Restarting capture… Relog your character after capture starts.";emit interfacesChanged();
        setCaptureReady(false);
        if(process.state()==QProcess::NotRunning) {startCapture();return;}
        restartingCapture=true;process.terminate();
        QTimer::singleShot(2000,this,[this] {if(restartingCapture&&process.state()!=QProcess::NotRunning) process.kill();});
    }
    QString backendPath;
    QString name, message="Starting capture…";
    double span=0;
    Bridge() {
        connect(&interfaceLister,&QProcess::errorOccurred,this,[this] {interfaceMessage=interfaceLister.errorString();emit interfacesChanged();});
        connect(&interfaceLister,qOverload<int,QProcess::ExitStatus>(&QProcess::finished),this,[this](int code,QProcess::ExitStatus) {
            if(code==0) {
                auto document=QJsonDocument::fromJson(interfaceLister.readAllStandardOutput());
                if(document.isArray()) {
                    interfaceRows=document.array().toVariantList();bool found=false;
                    for(const auto &row:interfaceRows) if(row.toMap().value("name").toString()==interfaceSetting) found=true;
                    if(!found) interfaceRows.append(QVariantMap{{"name",interfaceSetting},{"label",interfaceSetting+" (unavailable)"}});
                }
            } else interfaceMessage=QString::fromUtf8(interfaceLister.readAllStandardError()).trimmed();
            emit interfacesChanged();
        });
        connect(&updater,&QProcess::errorOccurred,this,[this] {updateMessage=updater.errorString();emit updateChanged();});
        connect(&updater,qOverload<int,QProcess::ExitStatus>(&QProcess::finished),this,[this](int code,QProcess::ExitStatus) {
            if(code!=0) {updateMessage=QString::fromUtf8(updater.readAllStandardError()).trimmed();availableUpdate=false;}
            else {auto info=QJsonDocument::fromJson(updater.readAllStandardOutput()).object();availableUpdate=info.value("available").toBool();downloadedPath=info.value("path").toString();updateMessage=!downloadedPath.isEmpty()?"Verified download saved. Close SlopMeter, then replace your installed copy.":availableUpdate?"Version "+info.value("latest").toString()+" is available.":"You have the latest compatible version.";}
            emit updateChanged();
        });
        QFile preview(":/preview.json");if(preview.open(QIODevice::ReadOnly)) previewParty=QJsonDocument::fromJson(preview.readAll()).array().toVariantList();
        publish.setInterval(pollInterval());
        connect(&publish,&QTimer::timeout,this,[this] {
            if(previewEnabled) refreshDisplay();
        });
        connect(&process,&QProcess::started,this,&Bridge::sendRefreshInterval);
        connect(&process,&QProcess::readyReadStandardOutput,this,[this] {
            buffer+=process.readAllStandardOutput();int end;
            while((end=buffer.indexOf('\n'))>=0) {
                auto line=buffer.left(end);buffer.remove(0,end+1);
                auto doc=QJsonDocument::fromJson(line);if(doc.isObject()) {
                    auto state=doc.object();
                    // History arrives only when changed. Preserve it even if coalescing drops a snapshot.
                    if(state.contains("history")) {history=state.value("history").toArray().toVariantList();emit historyChanged();}
                    pending=state;setCaptureReady(process.state()==QProcess::Running&&!replayMode);if(process.state()!=QProcess::NotRunning) setStatus(replayMode?"Replay running":state.value("character").toString().isEmpty()?"Capture active · waiting for character. Log out and back in with capture running.":"Capture running");}
            }
            // The backend already limits snapshots to the selected interval.
            // Apply the newest complete frame once per read, avoiding a second timer's latency.
            if(!pending.isEmpty()) {currentState=pending.toVariantMap();pending={};if(!previewEnabled) refreshDisplay();}
        });
        connect(&process,&QProcess::readyReadStandardError,this,[this] {auto text=QString::fromUtf8(process.readAllStandardError()).trimmed();if(!text.isEmpty()) setStatus(text);});
        connect(&process,&QProcess::errorOccurred,this,[this] {setCaptureReady(false);setStatus(process.errorString());});
        connect(&process,qOverload<int,QProcess::ExitStatus>(&QProcess::finished),this,[this](int code,QProcess::ExitStatus) {setCaptureReady(false);if(restartingCapture) {restartingCapture=false;QTimer::singleShot(0,this,[this]{startCapture();});return;}setStatus(code==0?"Capture finished":"Capture failed: "+message);});
    }
    ~Bridge() {
        restartingCapture=false;
        if(interfaceLister.state()!=QProcess::NotRunning) {interfaceLister.kill();interfaceLister.waitForFinished(1000);}
        if(updater.state()!=QProcess::NotRunning) {updater.kill();updater.waitForFinished(1000);}
        if(!scriptName.isEmpty()) {QDBusInterface scripting("org.kde.KWin","/Scripting","org.kde.kwin.Scripting",QDBusConnection::sessionBus());scripting.asyncCall("unloadScript",scriptName);}
        if(process.state()!=QProcess::NotRunning) {process.terminate();if(!process.waitForFinished(1500)) {process.kill();process.waitForFinished();}}}
    QString character() const {return name;}
    QString status() const {return message;}
    double duration() const {return span;}
    void setStatus(const QString &text) {if(message!=text) {message=text;emit statusChanged();}}
    QPoint position() const {return panel?QPoint(qRound(panel->x()),qRound(panel->y())):QPoint();}
    void positionOverlay(QPoint pos) {
        if(!panel) return;
        pos.setX(qBound(0,pos.x(),qMax(0,window->width()-qRound(panel->width()))));
        pos.setY(qBound(0,pos.y(),qMax(0,window->height()-qRound(panel->height()))));
        panel->setPosition(pos);updateInputMask();
    }
    Q_INVOKABLE void beginDrag() {dragOrigin=position();dragging=true;}
    Q_INVOKABLE void dragOverlay(int dx,int dy) {positionOverlay(dragOrigin+QPoint(dx,dy));}
    Q_INVOKABLE void endDrag() {dragging=false;savePosition();if(autoDisplay&&!gameScreen.isEmpty()) gameOutput(gameScreen,{});}
    void savePosition() {QSettings().setValue("position0",position());}
    Q_INVOKABLE void resetPosition() {positionOverlay({40,100});savePosition();window->show();}
    Q_INVOKABLE void hide() {window->hide();}
    Q_INVOKABLE void quit() {QCoreApplication::quit();}
signals:
    void interfacesChanged();
    void updateChanged();void captureChanged();void testModeChanged();void pollingChanged();void historyChanged();void detailChanged();void appearanceChanged();void characterChanged();void statusChanged();void durationChanged();void displayChanged();void encounterChanged();void bossChanged();
};
static void global(void *data,wl_registry*,uint32_t,const char *interface,uint32_t) {if(!std::strcmp(interface,"zwlr_layer_shell_v1")) *static_cast<bool*>(data)=true;}
static void removed(void*,wl_registry*,uint32_t) {}
int main(int argc,char **argv) {
    for(int i=1;i<argc;++i) if(std::strcmp(argv[i],"--version")==0) {printf("SlopMeter %s\n",SLOPMETER_VERSION);return 0;}

    bool headless=qEnvironmentVariable("QT_QPA_PLATFORM")=="offscreen";
    if(!headless) {
        auto display=wl_display_connect(nullptr);if(!display) {fprintf(stderr,"A Wayland session is required.\n");return 1;}
        bool supported=false;auto registry=wl_display_get_registry(display);const wl_registry_listener listener{global,removed};
        wl_registry_add_listener(registry,&listener,&supported);wl_display_roundtrip(display);wl_registry_destroy(registry);wl_display_disconnect(display);
        if(!supported) {fprintf(stderr,"Wayland layer shell is required.\n");return 1;}qputenv("QT_QPA_PLATFORM","wayland");
    }
    qunsetenv("QT_WAYLAND_SHELL_INTEGRATION");
    QApplication app(argc,argv);app.setQuitOnLastWindowClosed(false);app.setOrganizationName("AionDPS");app.setApplicationName("SlopMeter");
    std::unique_ptr<QTemporaryDir> selfTestSettings;
    if(app.arguments().contains("--ui-self-test")||app.arguments().contains("--screenshots")) {
        selfTestSettings=std::make_unique<QTemporaryDir>();if(!selfTestSettings->isValid()) return 1;
        QSettings::setDefaultFormat(QSettings::IniFormat);
        QSettings::setPath(QSettings::IniFormat,QSettings::UserScope,selfTestSettings->path());
    }
    if(!selfTestSettings) {
        QSettings previous("AionDPS","AionDPS"), current;
        if(!current.value("migration/aionDPS",false).toBool()) {
            for(const auto &key:previous.allKeys()) if(!current.contains(key)) current.setValue(key,previous.value(key));
            current.setValue("migration/aionDPS",true);
        }
    }
    Bridge bridge;bridge.headless=headless;
    QQmlApplicationEngine engine;engine.rootContext()->setContextProperty("backend",&bridge);engine.rootContext()->setContextProperty("actorModel",&bridge.actors);engine.rootContext()->setContextProperty("skillModel",&bridge.skills);
    engine.load(QUrl("qrc:/Main.qml"));if(engine.rootObjects().size()!=1) return 1;
    auto window=qobject_cast<QQuickWindow*>(engine.rootObjects().first());if(!window) return 1;bridge.window=window;
    bridge.panel=window->findChild<QQuickItem*>("meterPanel");if(!bridge.panel) return 1;
    QObject::connect(bridge.panel,&QQuickItem::xChanged,&bridge,&Bridge::updateInputMask);
    QObject::connect(bridge.panel,&QQuickItem::yChanged,&bridge,&Bridge::updateInputMask);
    QObject::connect(bridge.panel,&QQuickItem::opacityChanged,&bridge,&Bridge::updateInputMask);
    QObject::connect(bridge.panel,&QQuickItem::widthChanged,&bridge,[&bridge] {bridge.positionOverlay(bridge.position());});
    QObject::connect(bridge.panel,&QQuickItem::heightChanged,&bridge,[&bridge] {bridge.positionOverlay(bridge.position());});
    QStringList args=app.arguments().mid(1);bool test=args.removeAll("--ui-self-test")>0;
    QString screenshotDirectory;
    auto screenshotOption=args.indexOf("--screenshots");
    if(screenshotOption>=0) {if(screenshotOption+1>=args.size()) {fprintf(stderr,"--screenshots needs an output directory.\n");return 1;}screenshotDirectory=args.takeAt(screenshotOption+1);args.removeAt(screenshotOption);}
    bridge.detectionTest=args.removeAll("--ui-detection-test")>0;
    if(bridge.detectionTest) {
        bridge.startGameDetection();QTimer::singleShot(4000,&app,[&app] {fprintf(stderr,"Game-detection callback timed out.\n");app.exit(7);});return app.exec();
    }
    if(!headless) {
        auto layer=LayerShellQt::Window::get(window);layer->setLayer(LayerShellQt::Window::LayerOverlay);
        layer->setAnchors(LayerShellQt::Window::Anchors(LayerShellQt::Window::AnchorTop)|LayerShellQt::Window::AnchorBottom|LayerShellQt::Window::AnchorLeft|LayerShellQt::Window::AnchorRight);
#ifdef SLOPMETER_LAYER_DESIRED_SIZE
        layer->setDesiredSize(QSize(0,0));
#endif
        layer->setMargins(QMargins());
        layer->setCloseOnDismissed(false);layer->setExclusiveZone(-1);layer->setKeyboardInteractivity(LayerShellQt::Window::KeyboardInteractivityNone);
    }
    QObject::connect(window,&QQuickWindow::widthChanged,&bridge,[&bridge] {bridge.positionOverlay(bridge.position());});
    QObject::connect(window,&QQuickWindow::heightChanged,&bridge,[&bridge] {bridge.positionOverlay(bridge.position());});
    QObject::connect(window,&QWindow::screenChanged,&bridge,[&bridge](QScreen*) {emit bridge.displayChanged();});
    bridge.resizeCanvas();emit bridge.displayChanged();
    bridge.positionOverlay(QSettings().value("position0",QPoint(40,100)).toPoint());window->show();
    QPixmap icon(32,32);icon.fill(QColor("#19202d"));QPainter painter(&icon);painter.setPen(QColor("#9ad9ec"));painter.drawText(icon.rect(),Qt::AlignCenter,"DPS");painter.end();
    QSystemTrayIcon tray{QIcon(icon)};tray.setToolTip("SlopMeter");QMenu menu;
    menu.addAction("Show meter",window,[window] {window->show();});menu.addAction("Hide meter",window,[window] {window->hide();});
    auto displays=menu.addMenu("Display");
    displays->addAction("Follow AION 2",&bridge,[&bridge] {bridge.autoDisplay=true;bridge.gameOutput(bridge.gameScreen,{});});
    for(auto screen:app.screens()) {const auto name=screen->name();displays->addAction(name,&bridge,[&bridge,name] {bridge.autoDisplay=false;for(auto screen:QGuiApplication::screens()) if(screen->name()==name) {bridge.selectScreen(screen);break;}});}
    menu.addAction("Settings",window,[window] {QMetaObject::invokeMethod(window,"openSettings");});
    menu.addAction("Reset position",&bridge,&Bridge::resetPosition);menu.addSeparator();menu.addAction("Quit",&bridge,&Bridge::quit);
    tray.setContextMenu(&menu);QObject::connect(&tray,&QSystemTrayIcon::activated,&bridge,[window](QSystemTrayIcon::ActivationReason r) {if(r==QSystemTrayIcon::Trigger) window->setVisible(!window->isVisible());});tray.show();
    if(!test||!headless) bridge.startGameDetection();
    auto executable=qEnvironmentVariable("SLOPMETER_BACKEND");
    if(executable.isEmpty()) executable=qEnvironmentVariable("AIONDPS_BACKEND");
    if(executable.isEmpty()) {
        executable=app.applicationDirPath()+"/slopmeter-capture";
        if(!QFile::exists(executable)) executable=app.applicationDirPath()+"/../slopmeter-capture";
    }
    bridge.backendPath=executable;
    for(const auto &arg:args) if(arg=="-read"||arg=="--read"||arg.startsWith("-read=")||arg.startsWith("--read=")) bridge.replayMode=true;
    if(!screenshotDirectory.isEmpty()) {
        if(headless) {fprintf(stderr,"Documentation screenshots require a real Wayland desktop.\n");return 1;}
        bridge.setBarWidth(440);bridge.setBarHeight(44);bridge.setBarSpacing(6);bridge.setBackgroundOpacity(100);bridge.setTestMode(true);
        QDir().mkpath(screenshotDirectory);
        QTimer::singleShot(1400,&app,[&,screenshotDirectory] {
            auto grab=bridge.panel->grabToImage();
            QObject::connect(grab.get(),&QQuickItemGrabResult::ready,&app,[&,grab,screenshotDirectory] {
                if(!grab->image().save(screenshotDirectory+"/meter.png")) {app.exit(46);return;}
                const auto actor=bridge.displayPlayers().first().toMap().value("id").toUInt();
                QMetaObject::invokeMethod(window,"openDetails",Q_ARG(QVariant,QVariant(actor)));
                QTimer::singleShot(700,&app,[&,screenshotDirectory] {
                    for(auto candidate:QGuiApplication::allWindows()) if(candidate->objectName()=="fightDetailsWindow") {
                        auto details=qobject_cast<QQuickWindow*>(candidate);details->resize(1400,740);
                        QTimer::singleShot(500,&app,[&,details,screenshotDirectory] {
                            if(!details->grabWindow().save(screenshotDirectory+"/damage-breakdown.png")) {app.exit(47);return;}
                            fprintf(stderr,"Native Wayland screenshots saved.\n");app.quit();
                        });return;
                    }
                    app.exit(48);
                });
            });
        });
        return app.exec();
    }
    args.prepend("-json");args.append("-interval");args.append(QString::number(bridge.pollInterval())+"ms");for(int i=0;i<args.size();++i) if(args[i]=="-interface"&&i+1<args.size()) {bridge.setCaptureInterface(args[i+1]);args.removeAt(i+1);args.removeAt(i);break;}
    bridge.captureArguments=args;bridge.startCapture();bridge.refreshInterfaces();
    if(test) QTimer::singleShot(500,&app,[&] {
        if(QGuiApplication::allWindows().size()!=1) {app.exit(2);return;}
        bridge.hide();window->show();if(!window->isVisible()) {app.exit(3);return;}
        ActorModel model;
        int resets=0,insertions=0;
        QObject::connect(&model,&QAbstractItemModel::modelReset,&app,[&] {++resets;});
        QObject::connect(&model,&QAbstractItemModel::rowsInserted,&app,[&](const QModelIndex&,int,int) {++insertions;});
        QJsonArray rows{QJsonObject{{"id",1},{"name","Self"},{"damage",100},{"dps",100},{"share",40}},QJsonObject{{"id",2},{"name","Party"},{"damage",150},{"dps",150},{"share",60}}};
        model.update(rows);
        QElapsedTimer timing;timing.start();
        for(int i=0;i<10000;++i) model.update(rows);
        auto elapsed=timing.elapsed();
        QJsonArray reordered{rows[1],rows[0]};model.update(reordered);
        if(resets!=0||insertions!=2||model.rows[0].value("id").toInt()!=2) {app.exit(5);return;}
        fprintf(stderr,"Model check: 10000 stable updates in %lldms; no model resets.\n",elapsed);
        const auto nativePosition=window->position();const auto nativeSize=window->size();
        const auto pos=bridge.position();bridge.beginDrag();bridge.dragOverlay(pos.x()>12?-12:12,pos.y()>8?-8:8);
        if(bridge.position()==pos) {app.exit(4);return;}bridge.dragging=false;bridge.positionOverlay(pos);
        if(window->position()!=nativePosition||window->size()!=nativeSize||window->mask()!=QRegion(QRectF(bridge.panel->x(),bridge.panel->y(),bridge.panel->width(),bridge.panel->height()).toAlignedRect())) {app.exit(8);return;}
        if(!headless&&!bridge.gameScreen.isEmpty()&&window->screen()->name()!=bridge.gameScreen) {app.exit(9);return;}
        // Check content sizing, uniform settings, and bounds while keeping the native surface still.
        bridge.publish.stop();
        bridge.bossState.clear();emit bridge.bossChanged();
        auto settle=[&] {QEventLoop frames;QTimer::singleShot(60,&frames,&QEventLoop::quit);window->requestUpdate();frames.exec();};
        bridge.actors.update({});settle();const auto emptyHeight=bridge.panel->height();
        QJsonArray partyRows;
        for(int i=0;i<4;++i) partyRows.append(QJsonObject{{"id",i+1},{"name",QString("Player %1").arg(i+1)},{"class",QStringList{"Templar","Sorcerer","Cleric","Assassin"}[i]},{"damage",400-i*80},{"dps",200-i*40},{"fill",100-i*20},{"share",25}});
        bridge.actors.update(partyRows);settle();const auto partyHeight=bridge.panel->height();
        auto bars=window->findChild<QQuickItem*>("playerBars");
        if(!bars||partyHeight<=emptyHeight||qAbs(bars->height()-(4*bridge.barHeight()+3*bridge.barSpacing()))>1) {fprintf(stderr,"Content sizing failed: empty=%f party=%f bars=%f\n",emptyHeight,partyHeight,bars?bars->height():-1);app.exit(10);return;}
        if(!QMetaObject::invokeMethod(window,"openSettings")) {app.exit(11);return;}
        settle();
        QQuickWindow *settingsWindow=nullptr;
        for(auto candidate:QGuiApplication::allWindows()) if(candidate->objectName()=="settingsWindow") settingsWindow=qobject_cast<QQuickWindow*>(candidate);
        if(!settingsWindow||!settingsWindow->isVisible()||(settingsWindow->flags()&Qt::WindowDoesNotAcceptFocus)||qAbs(bridge.panel->height()-partyHeight)>1||window->size()!=nativeSize) {app.exit(12);return;}
        auto moveSlider=[&](const char *name,int value) {
            auto slider=settingsWindow->findChild<QObject*>(name);
            return slider&&slider->setProperty("value",value)&&QMetaObject::invokeMethod(slider,"moved");
        };
        if(bridge.backgroundOpacity()!=75||bridge.overallOpacity()!=100) {app.exit(21);return;}
        if(!moveSlider("barWidthSlider",440)||!moveSlider("barHeightSlider",32)||!moveSlider("barSpacingSlider",10)) {app.exit(13);return;}
        settle();
        if(qAbs(bars->height()-(4*32+3*10))>1||qAbs(bridge.panel->width()-qMin(468,window->width()))>1) {app.exit(13);return;}
        auto styleSelector=settingsWindow->findChild<QObject*>("barStyleSelector");
        if(!styleSelector||bridge.barStyle()!=0) {app.exit(42);return;}
        for(int style=0;style<4;++style) {
            styleSelector->setProperty("currentIndex",style);
            if(!QMetaObject::invokeMethod(styleSelector,"activated",Q_ARG(int,style))||bridge.barStyle()!=style) {app.exit(43);return;}
            settle();Bridge restored;if(restored.barStyle()!=style) {app.exit(44);return;}
            const auto path=qEnvironmentVariable("AIONDPS_TEST_SCREENSHOT");
            if(!path.isEmpty()) window->grabWindow().save(path+QString("-style-%1.png").arg(style));
        }
        bridge.setBarStyle(0);
        QQuickItem *compact=nullptr;
        QList<QQuickItem*> pending{bars};
        while(!pending.isEmpty()) {auto item=pending.takeFirst();if(item->objectName()=="compactShare") {compact=item;break;}pending.append(item->childItems());}
        if(!compact||!compact->isVisible()) {app.exit(45);return;}
        auto hpBar=window->findChild<QQuickItem*>("targetHealthBar");
        const auto savedBoss=bridge.bossState;bridge.bossState.clear();emit bridge.bossChanged();settle();
        if(!hpBar||hpBar->isVisible()) {app.exit(46);return;}
        bridge.bossState=savedBoss;emit bridge.bossChanged();settle();
        if(!moveSlider("backgroundOpacitySlider",20)) {app.exit(22);return;}
        settle();
        if(qAbs(bridge.panel->property("color").value<QColor>().alphaF()-0.2)>0.01||bridge.panel->opacity()!=1||bars->opacity()!=1) {app.exit(22);return;}
        if(!moveSlider("overallOpacitySlider",45)) {app.exit(23);return;}
        settle();
        if(qAbs(bridge.panel->opacity()-0.45)>0.01||settingsWindow->opacity()!=1||!settingsWindow->isVisible()||qAbs(bridge.panel->property("color").value<QColor>().alphaF()-0.2)>0.01) {app.exit(23);return;}
        if(!moveSlider("overallOpacitySlider",0)) {app.exit(24);return;}
        settle();
        if(window->mask()!=QRegion(QRect(-1,-1,1,1))||!settingsWindow->isVisible()) {app.exit(24);return;}
        settingsWindow->close();
        QMetaObject::invokeMethod(window,"openSettings");settle();
        if(!settingsWindow->isVisible()||QGuiApplication::allWindows().size()!=2) {app.exit(25);return;}
        // Values persisted by actual slider signals can be loaded by a fresh bridge.
        {Bridge restored;if(restored.barWidth()!=440||restored.barHeight()!=32||restored.barSpacing()!=10||restored.backgroundOpacity()!=20||restored.overallOpacity()!=0) {app.exit(26);return;}}
        bridge.resetAppearance();settle();
        if(bridge.backgroundOpacity()!=75||bridge.overallOpacity()!=100||qAbs(bridge.panel->height()-partyHeight)>1||window->mask()!=QRegion(QRectF(bridge.panel->x(),bridge.panel->y(),bridge.panel->width(),bridge.panel->height()).toAlignedRect())) {app.exit(27);return;}
        auto screenshot=qEnvironmentVariable("AIONDPS_TEST_SCREENSHOT");
        if(!screenshot.isEmpty()) {window->grabWindow().save(screenshot);settingsWindow->grabWindow().save(screenshot+"-settings.png");}
        // Preview uses the same live delegates and details models, never real history.
        const auto realState=bridge.currentState;const auto realHistory=bridge.history;
        // Idle clearing retains the archived fight, but the next session replaces it.
        if(!realHistory.isEmpty()) {
            const auto archived=realHistory.first().toMap().value("snapshot").toMap();
            auto idle=realState;idle["session"]=archived.value("session");idle["actors"]=QVariantList{};idle["active"]=false;idle["character"]="Current character";
            bridge.currentState=idle;bridge.refreshDisplay();
            if(bridge.displayPlayers().isEmpty()||bridge.displayState!=archived||bridge.character()!="Current character") {app.exit(39);return;}
            idle["session"]=archived.value("session").toULongLong()+1;idle["active"]=true;idle["actors"]=realState.value("actors");
            bridge.currentState=idle;bridge.refreshDisplay();
            if(bridge.displayState!=idle) {app.exit(40);return;}
            bridge.currentState=realState;bridge.refreshDisplay();
        }
        auto captureDot=window->findChild<QQuickItem*>("captureIndicator");
        auto displayText=window->findChild<QObject*>("displayLabel");
        if(!captureDot||!displayText||!displayText->property("text").toString().startsWith("Display - ")||bridge.captureActive()) {app.exit(41);return;}
        int peakPreviewRows=bridge.actors.rowCount();QObject::connect(&bridge.actors,&QAbstractItemModel::rowsInserted,&app,[&](const QModelIndex&,int,int) {peakPreviewRows=qMax(peakPreviewRows,bridge.actors.rowCount());});
        auto toggle=[&](const char *name,bool enabled) {
            auto control=settingsWindow->findChild<QObject*>(name);
            return control&&control->setProperty("checked",enabled)&&QMetaObject::invokeMethod(control,"toggled");
        };
        if(bridge.pollInterval()!=200||!bridge.showBorder()) {app.exit(28);return;}
        if(!toggle("testModeToggle",true)) {app.exit(29);return;}
        settle();
        if(!bridge.testMode()||bridge.actors.rowCount()!=5||peakPreviewRows>5||bridge.character()!="Test party"||bridge.skills.rowCount()!=8) {app.exit(29);return;}
        auto previewHealth=window->findChild<QQuickItem*>("targetHealthBar");
        if(!previewHealth||!previewHealth->isVisible()||!bridge.boss().value("maxKnown").toBool()||bridge.boss().value("percent").toDouble()<=0||bridge.boss().value("percent").toDouble()>=100) {app.exit(49);return;}
        if(!toggle("borderToggle",false)||QQmlProperty(bridge.panel,"border.width").read().toInt()!=0) {app.exit(30);return;}
        if(!moveSlider("barWidthSlider",480)||!moveSlider("barHeightSlider",40)||!moveSlider("barSpacingSlider",8)) {app.exit(31);return;}
        settle();
        if(qAbs(bars->height()-(5*40+4*8))>1||qAbs(bridge.panel->width()-qMin(508,window->width()))>1) {app.exit(31);return;}
        if(!moveSlider("pollIntervalSlider",50)||bridge.publish.interval()!=50) {app.exit(32);return;}
        settle();
        auto warning=settingsWindow->findChild<QQuickItem*>("cpuWarning");
        if(!warning||!warning->isVisible()) {app.exit(32);return;}
        const auto damageBefore=bridge.displayPlayers().first().toMap().value("damage").toLongLong();settle();
        if(bridge.displayPlayers().first().toMap().value("damage").toLongLong()<=damageBefore||bridge.currentState!=realState||bridge.history!=realHistory) {app.exit(33);return;}
        if(!moveSlider("pollIntervalSlider",1000)||bridge.publish.interval()!=1000) {app.exit(34);return;}
        if(warning->isVisible()) {app.exit(34);return;}
        {Bridge restored;if(restored.pollInterval()!=1000||restored.showBorder()||restored.testMode()) {app.exit(35);return;}}
        for(int rate=50;rate<=1000;rate+=50) {bridge.setPollInterval(rate);if(bridge.pollInterval()!=rate) {app.exit(36);return;}}
        bridge.setPollInterval(200);
        {
            auto selector=settingsWindow->findChild<QObject*>("captureInterfaceSelector");
            if(!selector||selector->property("enabled").toBool()) {app.exit(50);return;}
            bridge.interfaceRows={QVariantMap{{"name",""},{"label","Automatic"}},QVariantMap{{"name","any"},{"label","All interfaces"}}};emit bridge.interfacesChanged();settle();
            if(selector->property("currentIndex").toInt()!=0) {app.exit(56);return;}
            bridge.interfaceSetting="any";emit bridge.interfacesChanged();settle();
            if(selector->property("currentIndex").toInt()!=1) {app.exit(57);return;}
            bridge.interfaceSetting="";emit bridge.interfacesChanged();
            Bridge capture;
            capture.backendPath="/bin/sh";
            capture.captureArguments={"-c","trap 'exit 0' TERM; while :; do sleep 0.05; done","capture-test"};
            capture.setCaptureInterface("");capture.startCapture();
            if(!capture.process.waitForStarted(1000)||capture.process.arguments().contains("-interface")) {app.exit(51);return;}
            const auto initialPID=capture.process.processId();capture.applyCaptureInterface();
            if(capture.restartingCapture||capture.process.processId()!=initialPID) {app.exit(58);return;}
            capture.setCaptureInterface("any");
            {Bridge restored;if(restored.captureInterface()!="any") {app.exit(52);return;}}
            if(capture.process.arguments().contains("-interface")) {app.exit(53);return;}
            QEventLoop restarted;
            QObject::connect(&capture.process,&QProcess::started,&restarted,&QEventLoop::quit);
            QTimer::singleShot(3500,&restarted,&QEventLoop::quit);
            capture.applyCaptureInterface();restarted.exec();
            if(capture.restartingCapture||capture.process.state()!=QProcess::Running||capture.process.arguments().last()!="any") {app.exit(54);return;}
            capture.replayMode=true;capture.setCaptureInterface("lo");capture.applyCaptureInterface();
            if(capture.captureInterface()!="any"||capture.restartingCapture) {app.exit(55);return;}
            capture.replayMode=false;capture.setCaptureInterface("");
            fprintf(stderr,"Network interface persistence, explicit apply, restart and replay isolation checks passed.\n");
        }
        if(!screenshot.isEmpty()) {
            window->grabWindow().save(screenshot+"-preview.png");settingsWindow->grabWindow().save(screenshot+"-preview-settings.png");
            auto capture=bridge.panel->grabToImage();QEventLoop rendered;
            QObject::connect(capture.get(),&QQuickItemGrabResult::ready,&rendered,&QEventLoop::quit);QTimer::singleShot(1000,&rendered,&QEventLoop::quit);rendered.exec();
            capture->image().save(screenshot+"-meter.png");
        }
        if(!toggle("testModeToggle",false)) {app.exit(37);return;}
        if(bridge.currentState!=realState||bridge.history!=realHistory||bridge.displayState!=realState||bridge.publish.isActive()) {app.exit(37);return;}
        bridge.resetAppearance();settle();
        if(!bridge.showBorder()||QQmlProperty(bridge.panel,"border.width").read().toInt()!=1) {app.exit(38);return;}
        settingsWindow->close();
        // Historical selection and live detail models use the same data, with no resets.
        if(!bridge.history.isEmpty()) {
            const auto archived=bridge.history.first().toMap().value("snapshot").toMap();
            bridge.selectFight(archived.value("session").toULongLong());
            if(bridge.detailFight().value("session")!=archived.value("session")||bridge.displayPlayers().isEmpty()) {app.exit(14);return;}
            bridge.selectFight(0);
        }
        if(!bridge.displayPlayers().isEmpty()) {
            const auto actor=bridge.displayPlayers().first().toMap().value("id").toUInt();
            bridge.selectPlayer(actor);
            if(bridge.skills.rowCount()==0) {app.exit(15);return;}
            int skillResets=0;QObject::connect(&bridge.skills,&QAbstractItemModel::modelReset,&app,[&] {++skillResets;});
            for(int i=0;i<1000;++i) bridge.updateDetail();
            if(skillResets) {app.exit(16);return;}
            if(!QMetaObject::invokeMethod(window,"openDetails",Q_ARG(QVariant,QVariant(actor)))) {app.exit(17);return;}
            settle();
            QQuickWindow *detailWindow=nullptr;
            for(auto candidate:QGuiApplication::allWindows()) if(candidate->objectName()=="fightDetailsWindow") detailWindow=qobject_cast<QQuickWindow*>(candidate);
            if(!detailWindow||!detailWindow->isVisible()||(detailWindow->flags()&Qt::WindowDoesNotAcceptFocus)||window->size()!=nativeSize) {app.exit(18);return;}
            if(!screenshot.isEmpty()) detailWindow->grabWindow().save(screenshot+"-details.png");
            detailWindow->close();
            if(detailWindow->isVisible()||!window->isVisible()) {app.exit(19);return;}
            QMetaObject::invokeMethod(window,"openDetails",Q_ARG(QVariant,QVariant(actor)));settle();
            if(!detailWindow->isVisible()||QGuiApplication::allWindows().size()!=3) {app.exit(20);return;}
            detailWindow->close();
        }
        fprintf(stderr,"Preview isolation, live layout, polling, border, opacity and secondary-window lifecycle checks passed on %s.\n",qPrintable(window->screen()->name()));app.quit();
    });
    return app.exec();
}
#include "main.moc"
