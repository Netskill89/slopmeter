#pragma once

#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QRegularExpression>
#include <QTimer>

struct ReleaseVersion {
    QStringList core, prerelease;
    bool valid=false;
    explicit ReleaseVersion(const QString &text) {
        static const QRegularExpression pattern(
            "^v?(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\\.[0-9A-Za-z-]+)*))?(?:\\+[0-9A-Za-z-]+(?:\\.[0-9A-Za-z-]+)*)?$");
        const auto match=pattern.match(text);
        if(!match.hasMatch()) return;
        core={match.captured(1),match.captured(2),match.captured(3)};
        if(!match.captured(4).isEmpty()) prerelease=match.captured(4).split('.');
        for(const auto &part:prerelease) if(numeric(part)&&part.size()>1&&part.startsWith('0')) return;
        valid=true;
    }
    static bool numeric(const QString &text) {
        for(const auto c:text) if(c<'0'||c>'9') return false;
        return !text.isEmpty();
    }
    static int numberCompare(const QString &a,const QString &b) {
        if(a.size()!=b.size()) return a.size()>b.size()?1:-1;
        return QString::compare(a,b,Qt::CaseSensitive);
    }
    int compare(const ReleaseVersion &other) const {
        for(int i=0;i<3;++i) if(auto difference=numberCompare(core[i],other.core[i])) return difference;
        if(prerelease.isEmpty()!=other.prerelease.isEmpty()) return prerelease.isEmpty()?1:-1;
        for(int i=0;i<qMin(prerelease.size(),other.prerelease.size());++i) {
            const auto &a=prerelease[i],&b=other.prerelease[i];
            const bool an=numeric(a),bn=numeric(b);
            if(an!=bn) return an?-1:1;
            if(auto difference=an?numberCompare(a,b):QString::compare(a,b,Qt::CaseSensitive)) return difference;
        }
        return prerelease.size()==other.prerelease.size()?0:prerelease.size()>other.prerelease.size()?1:-1;
    }
};

class ReleaseCheck : public QObject {
    Q_OBJECT
    Q_PROPERTY(bool available READ available NOTIFY changed)
    Q_PROPERTY(QString version READ version NOTIFY changed)
    Q_PROPERTY(QString url READ url CONSTANT)
public:
    explicit ReleaseCheck(const QString &current,QObject *parent=nullptr):QObject(parent),installed(current) {}
    bool available() const {return !newer.isEmpty();}
    QString version() const {return newer;}
    QString url() const {return "https://github.com/Netskill89/slopmeter/releases";}
    // Also used by offline regression tests; malformed replies preserve the last result.
    void applyReleases(const QByteArray &body) {
        QJsonParseError error;const auto document=QJsonDocument::fromJson(body,&error);
        const ReleaseVersion current(installed);
        if(error.error!=QJsonParseError::NoError||!document.isArray()||!current.valid) return;
        auto best=current;QString next;
        for(const auto &entry:document.array()) {
            const auto release=entry.toObject();const auto tag=release.value("tag_name").toString();
            const ReleaseVersion candidate(tag);
            if(!candidate.valid||release.value("draft").toBool()||release.value("published_at").toString().isEmpty()) continue;
            if(current.prerelease.isEmpty()&&(release.value("prerelease").toBool()||!candidate.prerelease.isEmpty())) continue;
            if(candidate.compare(best)>0) {best=candidate;next=tag.startsWith('v')?tag.mid(1):tag;}
        }
        if(next!=newer) {newer=next;emit changed();}
    }
    void start() {
        connect(&schedule,&QTimer::timeout,this,&ReleaseCheck::check);
        schedule.start(6*60*60*1000);QTimer::singleShot(1500,this,&ReleaseCheck::check);
    }
signals:
    void changed();
private:
    QString installed,newer;
    QNetworkAccessManager network;
    QTimer schedule;
    bool pending=false;
    void check() {
        if(pending) return;
        QNetworkRequest request(QUrl("https://api.github.com/repos/Netskill89/slopmeter/releases?per_page=100"));
        // Qt 6.11's HTTP/2 ready-read handler can read a closed TLS socket after
        // GitHub closes an idle connection. This small request needs no HTTP/2.
        request.setAttribute(QNetworkRequest::Http2AllowedAttribute,false);
        request.setRawHeader("Accept","application/vnd.github+json");
        request.setRawHeader("User-Agent",("SlopMeter/"+installed).toUtf8());
        request.setTransferTimeout(10000);
        pending=true;auto reply=network.get(request);
        QTimer::singleShot(15000,reply,[reply] {if(!reply->isFinished()) reply->abort();});
        connect(reply,&QNetworkReply::finished,this,[this,reply] {
            pending=false;
            if(reply->error()==QNetworkReply::NoError&&reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt()==200) applyReleases(reply->readAll());
            reply->deleteLater();
        });
    }
};
