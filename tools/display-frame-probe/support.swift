// Copied from the frozen dcf06eb window recorder; independent display tool.
import CryptoKit
import Darwin
import Foundation

struct ProbeError: Error, CustomStringConvertible {
  let description: String
  init(_ description: String) { self.description = description }
}

func validMarker(_ text: String) -> Bool {
  !text.isEmpty && text.utf8.count <= 32
    && text.utf8.allSatisfy {
      (48...57).contains($0) || (65...90).contains($0) || (97...122).contains($0) || $0 == 45
        || $0 == 95
    }
}

func upperBoundMilliseconds(
  marker: UInt64, displayed: UInt64, numerator: UInt32, denominator: UInt32
) -> Double? {
  guard marker > 0, displayed >= marker, numerator > 0, denominator > 0 else { return nil }
  return Double(displayed - marker) * Double(numerator) / Double(denominator) / 1_000_000
}

func sha256Hex(_ data: Data) -> String {
  SHA256.hash(data: data).map { String(format: "%02x", $0) }.joined()
}

func writeJSON(_ value: [String: Any], to url: URL) throws {
  let data = try JSONSerialization.data(
    withJSONObject: value, options: [.prettyPrinted, .sortedKeys])
  try data.write(to: url, options: .withoutOverwriting)
  try FileManager.default.setAttributes([.posixPermissions: 0o600], ofItemAtPath: url.path)
}

func createPrivateOutput(_ url: URL) throws {
  guard url.path.withCString({ mkdir($0, 0o700) }) == 0 else {
    throw ProbeError(
      "Cannot create a new private output directory (errno \(errno)); existing data is never reused"
    )
  }
}

// Foundation can abbreviate an existing /private/tmp to /tmp while keeping a
// missing child under /private/tmp. Resolve the longest existing POSIX ancestor
// instead, so aliases and symlinked parents have one physical prefix.
func physicalComponents(_ url: URL) -> [String]? {
  var path = url.path
  guard path.hasPrefix("/"),
    !path.split(separator: "/").contains(where: { $0 == "." || $0 == ".." })
  else { return nil }
  var tail: [String] = []
  while true {
    if let resolved = realpath(path, nil) {
      defer { free(resolved) }
      return String(cString: resolved).split(separator: "/").map(String.init) + tail.reversed()
    }
    var attributes = stat()
    guard lstat(path, &attributes) != 0, errno == ENOENT else { return nil }
    let name = (path as NSString).lastPathComponent
    guard !name.isEmpty, name != "/" else { return nil }
    tail.append(name)
    path = (path as NSString).deletingLastPathComponent
  }
}

func permittedOutput(_ url: URL, roots: [URL]) -> Bool {
  guard let path = physicalComponents(url) else { return false }
  guard !path.contains(where: { $0.lowercased().hasSuffix(".app") }) else { return false }
  return roots.contains { root in
    guard let prefix = physicalComponents(root), !prefix.isEmpty else { return false }
    return path.count > prefix.count && Array(path.prefix(prefix.count)) == prefix
  }
}
