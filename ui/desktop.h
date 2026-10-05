#pragma once

#include <QObject>
#include <functional>

// A second launch restores the existing overlay instead of starting another capture.
class DesktopControl : public QObject {
    Q_OBJECT
    Q_CLASSINFO("D-Bus Interface", "io.github.Netskill89.SlopMeter")
public:
    using QObject::QObject;
    std::function<bool()> meterVisible;
public slots:
    void Show() { emit restoreRequested(); }
    bool IsVisible() const { return meterVisible && meterVisible(); }
signals:
    void restoreRequested();
};
