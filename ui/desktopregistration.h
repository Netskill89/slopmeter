#pragma once

#include <QCoreApplication>
#include <QDir>
#include <QFile>
#include <QFileInfo>
#include <QSaveFile>
#include <QStandardPaths>

inline QString desktopExec(const QString &path) {
    QString escaped=path;
    escaped.replace('\\',"\\\\");escaped.replace('"',"\\\"");
    escaped.replace('`',"\\`");escaped.replace('$',"\\$");escaped.replace('%',"%%");
    // Desktop-entry string escaping precedes Exec argument parsing.
    escaped.replace('\\',"\\\\");
    return '"'+escaped+'"';
}

// Portable launches need an app identity for the host portal too. This hidden
// entry registers the identity; install.sh remains the visible menu installer.
inline bool registerDesktopIdentity() {
    if(!QStandardPaths::locate(QStandardPaths::GenericDataLocation,"applications/slopmeter.desktop").isEmpty()) return true;
    const auto root=QStandardPaths::writableLocation(QStandardPaths::GenericDataLocation);
    if(root.isEmpty()||!QDir().mkpath(root+"/applications")||!QDir().mkpath(root+"/icons/hicolor/scalable/apps")) return false;
    QString executable=qEnvironmentVariable("APPIMAGE");
    if(executable.isEmpty()||!QFileInfo(executable).isExecutable()) {
        const auto appRun=QDir(QCoreApplication::applicationDirPath()).absoluteFilePath("../../AppRun");
        executable=QFileInfo(appRun).isExecutable()?QDir::cleanPath(appRun):QCoreApplication::applicationFilePath();
    }
    QFile source(":/app/slopmeter.svg");QSaveFile icon(root+"/icons/hicolor/scalable/apps/slopmeter.svg");
    if(!source.open(QIODevice::ReadOnly)||!icon.open(QIODevice::WriteOnly)) return false;
    const auto image=source.readAll();if(icon.write(image)!=image.size()||!icon.commit()) return false;
    const auto entry=QString("[Desktop Entry]\nType=Application\nName=SlopMeter\nComment=AION 2 DPS meter for Linux Wayland\nExec=%1\nIcon=slopmeter\nTerminal=false\nNoDisplay=true\nCategories=Game;Utility;\nStartupWMClass=SlopMeter\n").arg(desktopExec(executable)).toUtf8();
    QSaveFile desktop(root+"/applications/slopmeter.desktop");
    return desktop.open(QIODevice::WriteOnly)&&desktop.write(entry)==entry.size()&&desktop.commit();
}
