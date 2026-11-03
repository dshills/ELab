import Cocoa

struct LaunchInfo: Decodable {
    let version: Int
    let kind: String
    let url: String
    let root: String
    let instance: String
    let token: String
    let owned: Bool
}

final class Launcher: NSObject, NSApplicationDelegate, NSWindowDelegate {
    var window: NSWindow!
    let status = NSTextField(wrappingLabelWithString: "Starting ELab…")
    let detail = NSTextField(wrappingLabelWithString: "Your projects are saved on this Mac.")
    let openButton = NSButton(title: "Open ELab", target: nil, action: nil)
    let retryButton = NSButton(title: "Try again", target: nil, action: nil)
    var child: Process?
    var helper: Process?
    var reply: URL!
    var temporary: URL!
    var logURL: URL!
    var logHandle: FileHandle!
    var info: LaunchInfo?
    var bootstrapTimer: Timer?
    var monitorTimer: Timer?
    var quitting = false
    var queuedQuit = false
    var startedAt = Date()
    var backend: URL!
    var root: URL!

    func applicationDidFinishLaunching(_ notification: Notification) {
        let fm = FileManager.default
        let override = Bundle.main.object(forInfoDictionaryKey: "ELabWebTestDataDirectory") as? String
        root = override.map { URL(fileURLWithPath: $0, isDirectory: true) }
            ?? fm.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0].appendingPathComponent("ELab/web", isDirectory: true)
        window = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 560, height: 410), styleMask: [.titled, .closable, .miniaturizable], backing: .buffered, defer: false)
        window.title = "ELab Web"
        window.delegate = self
        window.center()
        let heading = NSTextField(labelWithString: "ELab Web")
        heading.font = .systemFont(ofSize: 24, weight: .semibold)
        status.font = .systemFont(ofSize: 15, weight: .medium)
        detail.textColor = .secondaryLabelColor
        openButton.target = self; openButton.action = #selector(openWorkbench)
        retryButton.target = self; retryButton.action = #selector(start)
        let folder = NSButton(title: "Projects folder", target: self, action: #selector(showProjects))
        let logs = NSButton(title: "View log", target: self, action: #selector(showLog))
        let quit = NSButton(title: "Quit ELab", target: NSApp, action: #selector(NSApplication.terminate(_:)))
        let actions = NSStackView(views: [openButton, retryButton, quit])
        actions.orientation = .horizontal; actions.spacing = 8
        let support = NSStackView(views: [folder, logs])
        support.orientation = .horizontal; support.spacing = 8
        let footnote = NSTextField(wrappingLabelWithString: "Keep ELab running while using the browser. Quitting saves your projects; closing a browser tab leaves ELab running.")
        footnote.font = .systemFont(ofSize: 12); footnote.textColor = .secondaryLabelColor
        let stack = NSStackView(views: [heading, status, detail, actions, support, footnote])
        stack.orientation = .vertical; stack.alignment = .leading; stack.spacing = 14
        stack.translatesAutoresizingMaskIntoConstraints = false
        window.contentView!.addSubview(stack)
        NSLayoutConstraint.activate([
            stack.leadingAnchor.constraint(equalTo: window.contentView!.leadingAnchor, constant: 24),
            stack.trailingAnchor.constraint(equalTo: window.contentView!.trailingAnchor, constant: -24),
            stack.topAnchor.constraint(equalTo: window.contentView!.topAnchor, constant: 24)
        ])
        let menu = NSMenu()
        let appItem = NSMenuItem(); menu.addItem(appItem)
        let appMenu = NSMenu(); appItem.submenu = appMenu
        appMenu.addItem(withTitle: "Open ELab", action: #selector(openWorkbench), keyEquivalent: "o").target = self
        appMenu.addItem(.separator())
        appMenu.addItem(withTitle: "Quit ELab Web", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        NSApp.mainMenu = menu
        window.makeKeyAndOrderFront(nil); NSApp.activate(ignoringOtherApps: true)
        backend = Bundle.main.bundleURL.appendingPathComponent("Contents/Resources/elab-web")
        start()
    }
    func prepareStorage() throws {
        let fm = FileManager.default
        if logHandle == nil {
            let logsDir = root.appendingPathComponent("Logs", isDirectory: true)
            try fm.createDirectory(at: logsDir, withIntermediateDirectories: true, attributes: [.posixPermissions: 0o700])
            logURL = logsDir.appendingPathComponent("launcher-\(UUID().uuidString).log")
            guard fm.createFile(atPath: logURL.path, contents: nil, attributes: [.posixPermissions: 0o600]) else { throw CocoaError(.fileWriteUnknown) }
            logHandle = try FileHandle(forWritingTo: logURL)
        }
        if temporary == nil {
            let location = fm.temporaryDirectory.appendingPathComponent("elab-web-\(UUID().uuidString)", isDirectory: true)
            try fm.createDirectory(at: location, withIntermediateDirectories: false, attributes: [.posixPermissions: 0o700])
            temporary = location
        }
    }

    @objc func start() {
        guard helper == nil else { return }
        if child?.isRunning == true {
            if info != nil {openWorkbench()} else {
                startedAt = Date(); status.stringValue = "Waiting for ELab to start…"
                bootstrapTimer?.invalidate()
                bootstrapTimer = Timer.scheduledTimer(withTimeInterval: 0.1, repeats: true) { [weak self] _ in self?.readReply() }
            }
            return
        }
        do {try prepareStorage()} catch {failed("ELab could not prepare local storage: \(error.localizedDescription)");return}
        monitorTimer?.invalidate(); info = nil
        reply = temporary.appendingPathComponent("launch-\(UUID().uuidString).json")
        status.stringValue = "Starting ELab…"; detail.stringValue = "Your projects are saved on this Mac."
        openButton.isEnabled = false; retryButton.isHidden = true
        startedAt = Date()
        let process = Process(); process.executableURL = backend
        process.arguments = ["--auto-port", "--data-dir", root.path, "--startup-file", reply.path]
        process.standardOutput = logHandle; process.standardError = logHandle
        process.terminationHandler = { [weak self] p in DispatchQueue.main.async { self?.childExited(p) } }
        child = process
        do {
            try process.run()
            bootstrapTimer?.invalidate()
            bootstrapTimer = Timer.scheduledTimer(withTimeInterval: 0.1, repeats: true) { [weak self] _ in self?.readReply() }
        } catch { child = nil; failed("ELab could not start: \(error.localizedDescription)") }
    }
    func readReply() {
        guard info == nil, reply != nil else { return }
        if let data = try? Data(contentsOf: reply), let value = try? JSONDecoder().decode(LaunchInfo.self, from: data),
           value.version == 1, value.kind == "elab-web-host", let address = URL(string: value.url),
           address.scheme == "http", ["localhost", "127.0.0.1", "::1"].contains((address.host ?? "").trimmingCharacters(in: CharacterSet(charactersIn: "[]"))), address.port != nil,
           address.user == nil, address.password == nil, address.query == nil, address.fragment == nil {
            info = value; bootstrapTimer?.invalidate()
            status.stringValue = value.owned ? "ELab is running on this Mac." : "Using the running ELab server."
            detail.stringValue = value.url
            openButton.isEnabled = true; retryButton.isHidden = true
            if !quitting { openWorkbench() }
            if !value.owned {
                monitorTimer = Timer.scheduledTimer(withTimeInterval: 5, repeats: true) { [weak self] _ in self?.checkAttachedHost() }
            }
        } else if Date().timeIntervalSince(startedAt) > 30 {
            bootstrapTimer?.invalidate()
            failed("ELab is taking longer to start. Check the log before trying again.")
        }
    }
    func childExited(_ process: Process) {
        readReply()
        if quitting {
            if info == nil || process.terminationStatus == 0 {NSApp.reply(toApplicationShouldTerminate: true)}
            else {info = nil;quitting = false;failed("ELab stopped unexpectedly. Check the log before reopening.");NSApp.reply(toApplicationShouldTerminate: false)}
            return
        }
        if info?.owned == false && process.terminationStatus == 0 { return }
        bootstrapTimer?.invalidate(); monitorTimer?.invalidate()
        if process.terminationStatus == 0 { stopped() }
        else {
            info = nil
            let text = logURL.flatMap {try? String(contentsOf: $0, encoding: .utf8)} ?? ""
            if text.contains("another ELab server is using this project library") {
                failed("Another ELab server is using your projects. Close that server, then choose Try again.")
            } else {failed("ELab could not stay running. View the log for details, then try again.")}
        }
    }
    func failed(_ message: String) {
        status.stringValue = message; detail.stringValue = logURL?.path ?? root?.path ?? ""
        openButton.isEnabled = info != nil; retryButton.isHidden = false
    }
    func stopped() {
        monitorTimer?.invalidate(); status.stringValue = "ELab has stopped."
        detail.stringValue = "Your saved projects remain on this Mac."
        openButton.isEnabled = false; retryButton.isHidden = false; info = nil
    }
    @objc func openWorkbench() {
        guard let value = info, let address = URL(string: value.url) else { return }
        if !NSWorkspace.shared.open(address) { failed("The browser could not open ELab. Open \(value.url) in your browser.") }
    }
    @objc func showProjects() { if let root = root { NSWorkspace.shared.open(root) } }
    @objc func showLog() { if let logURL = logURL { NSWorkspace.shared.activateFileViewerSelecting([logURL]) } }
    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        window.makeKeyAndOrderFront(nil); openWorkbench(); return true
    }
    func windowShouldClose(_ sender: NSWindow) -> Bool { NSApp.terminate(nil); return false }
    func control(_ flag: String, completion: @escaping (Bool) -> Void) {
        guard helper == nil, let reply = reply else { completion(false); return }
        let process = Process(); process.executableURL = backend; process.arguments = [flag, reply.path]
        process.standardOutput = logHandle; process.standardError = logHandle
        helper = process
        process.terminationHandler = { [weak self] p in DispatchQueue.main.async { self?.helper = nil; completion(p.terminationStatus == 0) } }
        do { try process.run() } catch { helper = nil; completion(false) }
    }
    func checkAttachedHost() {
        guard !quitting, helper == nil else { return }
        control("--host-status") { [weak self] ok in
            guard let self = self else {return}
            if self.queuedQuit {
                self.queuedQuit = false
                NSApp.reply(toApplicationShouldTerminate: false)
                DispatchQueue.main.async {NSApp.terminate(nil)}
                return
            }
            if !ok {self.monitorTimer?.invalidate();self.failed("ELab's connection is unavailable. Try again or view the log.")}
        }
    }
    func applicationShouldTerminate(_ sender: NSApplication) -> NSApplication.TerminateReply {
        if quitting { return .terminateLater }
        if info == nil {
            if child?.isRunning == true { quitting = true; bootstrapTimer?.invalidate(); child?.terminate(); return .terminateLater }
            return .terminateNow
        }
        guard helper == nil else {
            queuedQuit = true; monitorTimer?.invalidate()
            status.stringValue = "Waiting for ELab before quitting…"
            return .terminateLater
        }
        quitting = true; monitorTimer?.invalidate()
        status.stringValue = "Saving projects and stopping ELab…"; openButton.isEnabled = false
        control("--stop-host") { [weak self] ok in
            guard let self = self else { return }
            if ok { NSApp.reply(toApplicationShouldTerminate: true) }
            else {
                self.quitting = false
                self.failed("ELab could not finish saving and stopping. It may still be running. Check the log and try quitting again.")
                NSApp.reply(toApplicationShouldTerminate: false)
                self.window.makeKeyAndOrderFront(nil)
            }
        }
        return .terminateLater
    }
    func applicationWillTerminate(_ notification: Notification) {
        bootstrapTimer?.invalidate(); monitorTimer?.invalidate()
        try? logHandle?.close()
        if let temporary = temporary { try? FileManager.default.removeItem(at: temporary) }
    }
}

let application = NSApplication.shared
let launcher = Launcher()
application.setActivationPolicy(.regular)
application.delegate = launcher
application.run()
