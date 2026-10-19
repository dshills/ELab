import CoreGraphics
import Foundation

struct Patch: Sendable {
  let id: String
  let rect: CGRect
  let bits: [String]
  let cell: Int
}

func calibrationPatches(_ window: CGRect) throws -> [Patch] {
  guard integralRectangle(window), window.width >= 160, window.height >= 160 else {
    throw ProbeError("Calibration target must have finite integral bounds of at least 160 points")
  }
  let x = window.minX
  let y = window.minY
  let w = window.width
  let h = window.height
  let patterns = [
    ("tl", CGRect(x: x + 2, y: y + 2, width: 16, height: 16), ["1100", "1010", "0111", "0001"]),
    (
      "tr", CGRect(x: x + w - 18, y: y + 2, width: 16, height: 16),
      ["1011", "0100", "1101", "0010"]
    ),
    (
      "bl", CGRect(x: x + 2, y: y + h - 18, width: 16, height: 16),
      ["0011", "1110", "1000", "0101"]
    ),
    (
      "br", CGRect(x: x + w - 18, y: y + h - 18, width: 16, height: 16),
      ["0110", "1001", "0010", "1101"]
    ),
  ]
  var result = patterns.map { Patch(id: $0.0, rect: $0.1, bits: $0.2, cell: 4) }
  // Away from the observed central cursor-shaped region; static occlusion only.
  let cx = x + floor(w / 2) - 48
  let cy = y + floor(h * 0.65) - 32
  let checker = (0..<8).map { r in
    String((0..<12).map { c in (r + c) % 2 == 0 ? Character("1") : Character("0") })
  }
  result.append(
    Patch(id: "occluder", rect: CGRect(x: cx, y: cy, width: 96, height: 64), bits: checker, cell: 8)
  )
  guard result.allSatisfy({ window.contains($0.rect) && integralRectangle($0.rect) }) else {
    throw ProbeError("Patch leaves the approved target rectangle")
  }
  for (i, a) in result.enumerated() {
    for b in result.dropFirst(i + 1) {
      guard !a.rect.intersects(b.rect) else { throw ProbeError("Calibration patches overlap") }
    }
  }
  return result
}

func appKitRect(_ global: CGRect, primaryHeight: Double) throws -> CGRect {
  guard integralRectangle(global), primaryHeight.isFinite, primaryHeight > 0,
    primaryHeight.rounded() == primaryHeight
  else {
    throw ProbeError("Invalid global/AppKit coordinate transform")
  }
  return CGRect(
    x: global.minX, y: primaryHeight - global.maxY, width: global.width, height: global.height)
}

func fixtureOfflineChecks() throws {
  let window = CGRect(x: 2034, y: 303, width: 1053, height: 802)
  let p = try calibrationPatches(window)
  guard p.count == 5, p[0].rect == CGRect(x: 2036, y: 305, width: 16, height: 16),
    p[3].rect == CGRect(x: 3069, y: 1087, width: 16, height: 16),
    p[4].rect == CGRect(x: 2512, y: 792, width: 96, height: 64),
    try appKitRect(p[0].rect, primaryHeight: 1440)
      == CGRect(x: 2036, y: 1119, width: 16, height: 16)
  else { throw ProbeError("Independent pilot layout expectations differ") }
  let negative = try appKitRect(
    CGRect(x: -500, y: -200, width: 16, height: 16), primaryHeight: 1440)
  guard negative == CGRect(x: -500, y: 1624, width: 16, height: 16) else {
    throw ProbeError("Negative display origin transform differs")
  }
  for bad in [
    CGRect.zero, CGRect(x: 0, y: 0, width: 159, height: 802),
    CGRect(x: 0.5, y: 0, width: 1053, height: 802),
    CGRect(x: Double.nan, y: 0, width: 1053, height: 802),
  ] {
    var refused = false
    do { _ = try calibrationPatches(bad) } catch { refused = true }
    guard refused else { throw ProbeError("Invalid calibration rectangle admitted") }
  }
  guard Set(p.map { $0.id }).count == 5, Set(p.prefix(4).map { $0.bits.joined() }).count == 4,
    p.allSatisfy({ patch in
      patch.bits.count * patch.cell == Int(patch.rect.height)
        && patch.bits.allSatisfy { line in
          line.count * patch.cell == Int(patch.rect.width)
            && line.allSatisfy { $0 == "0" || $0 == "1" }
        }
    })
  else { throw ProbeError("Pattern identities/dimensions differ") }
  print(
    "{\"offline_checks_passed\":true,\"layout_cases\":7,\"ui_created\":false,\"capture_executed\":false}"
  )
}
