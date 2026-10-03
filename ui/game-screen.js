// Loaded for this process only; no permanent KWin configuration changes.
const watched = new Set();
function isGame(w) {
    const application = [w.resourceClass, w.resourceName].join(" ");
    if (/(?:^|[^a-z0-9])aion[\s_-]*2(?:\.exe|[^a-z0-9]|$)/i.test(application)) return true;
    return /(?:steam_app_\d+|wine)/i.test(application) && /^aion[\s_-]*2(?:\b|$)/i.test(w.caption);
}
function report() {
    const games = workspace.windowList().filter(isGame);
    const game = games.find(w => w.active) || games.find(w => !w.minimized) || games[0];
    callDBus("@SERVICE@", "/AionDPS", "org.aiondps.Screen", "gameOutput",
             game && game.output ? game.output.name : "",
             game ? game.caption : "");
}
function watch(w) {
    if (watched.has(w)) return;
    watched.add(w);
    w.outputChanged.connect(report);
    w.captionChanged.connect(report);
    w.minimizedChanged.connect(report);
}
workspace.windowList().forEach(watch);
workspace.windowAdded.connect(w => { watch(w); report(); });
workspace.windowRemoved.connect(report);
workspace.windowActivated.connect(w => { if(w) watch(w); report(); });
report();
