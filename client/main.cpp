#include <QDebug>
#include <QTimer>
#ifdef DEFIGURU_REVIEW_CAPTURE
#include <QImage>
#include <QQuickWindow>
#include <QSettings>
#endif
#include <libssh/libssh.h>
#include <openssl/ssl.h>

#include "amneziaApplication.h"
#include "core/utils/osSignalHandler.h"
#include "core/utils/migrations.h"
#include "core/utils/appUiConfig.h"
#include "version.h"


// use openssl symbols to prevent linker throwing-off the OpenSSL dependency
void anchorOpenSSL() {
    SSL_CTX_free(SSL_CTX_new(TLS_method()));
}

#ifdef Q_OS_WIN
    #include "Windows.h"
#endif

#if defined(Q_OS_IOS)
    #include "platforms/ios/QtAppDelegate-C-Interface.h"
#endif

#if !defined(Q_OS_ANDROID) && !defined(Q_OS_IOS) && !defined(MACOS_NE)
bool isAnotherInstanceRunning()
{
    QLocalSocket socket;
    socket.connectToServer(APP_INSTANCE_NAME);
    if (socket.waitForConnected(500)) {
        qWarning() << APPLICATION_NAME << "is already running";
        return true;
    }
    return false;
}
#endif

int main(int argc, char *argv[])
{
#ifdef DEFIGURU_REVIEW_CAPTURE
    const QString capturePath = qEnvironmentVariable("DEFIGURU_REVIEW_CAPTURE_PATH");
    const QString captureSettings = qEnvironmentVariable("DEFIGURU_REVIEW_SETTINGS_DIR");
    const bool captureReview = !capturePath.isEmpty() && !captureSettings.isEmpty();
    if (captureReview) {
        QSettings::setDefaultFormat(QSettings::IniFormat);
        QSettings::setPath(QSettings::IniFormat, QSettings::UserScope, captureSettings);
    }
#endif
    Migrations migrationsManager;
    migrationsManager.doMigrations();

#ifdef Q_OS_WIN
    AllowSetForegroundWindow(ASFW_ANY);
#endif

#ifdef Q_OS_ANDROID
    // QTBUG-95974 QTBUG-95764 QTBUG-102168
    qputenv("QT_ANDROID_DISABLE_ACCESSIBILITY", "1");
    qputenv("ANDROID_OPENSSL_SUFFIX", "_3");
#endif

    AmneziaApplication app(argc, argv);
    OsSignalHandler::setup();

    anchorOpenSSL();

    ssh_init();
    QObject::connect(&app, &QCoreApplication::aboutToQuit, []() {
        ssh_finalize();
    });

#if !defined(Q_OS_ANDROID) && !defined(Q_OS_IOS) && !defined(MACOS_NE)
    if (isAnotherInstanceRunning()) {
        QTimer::singleShot(1000, &app, [&]() { app.quit(); });
        return app.exec();
    }
    app.startLocalServer();
#endif

// Allow to raise app window if secondary instance launched
#ifdef Q_OS_WIN
    AllowSetForegroundWindow(0);
#endif

    app.registerTypes();

    app.setApplicationName(APPLICATION_NAME);
    app.setOrganizationName(ORGANIZATION_NAME);
    app.setApplicationDisplayName(APPLICATION_NAME);

    app.loadFonts();

    bool doExec = app.parseCommands();

    if (doExec) {
        app.init();
#ifdef DEFIGURU_REVIEW_CAPTURE
        if (captureReview) {
            // Actual QML render from an empty review settings context. Never
            // imports a connection, clicks Connect or requests VPN permission.
            QTimer::singleShot(2500, &app, [&app, capturePath]() {
                bool saved = false;
                for (QObject *object : app.qmlEngine()->rootObjects()) {
                    if (auto *window = qobject_cast<QQuickWindow *>(object)) {
                        const QImage frame = window->grabWindow();
                        saved = !frame.isNull() && frame.save(capturePath, "PNG");
                        break;
                    }
                }
                app.exit(saved ? 0 : 2);
            });
            QTimer::singleShot(15000, &app, [&app]() { app.exit(3); });
        }
#endif

        qInfo().noquote() << QString("Started %1 version %2 %3").arg(APPLICATION_NAME, APP_VERSION, GIT_COMMIT_HASH);
        qInfo().noquote() << QString("%1 (%2)").arg(QSysInfo::prettyProductName(), QSysInfo::currentCpuArchitecture());

        return app.exec();
    }
    return 0;
}
