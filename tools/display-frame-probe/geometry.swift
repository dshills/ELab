import CoreGraphics
import Foundation

func recordedNumber(_ value: Double?) -> Any {
  guard let value else { return NSNull() }
  return value.isFinite ? NSNumber(value: value) : String(describing: value)
}

func rectangle(_ r: CGRect) -> [String: Any] {
  [
    "x": recordedNumber(r.origin.x), "y": recordedNumber(r.origin.y),
    "width": recordedNumber(r.size.width), "height": recordedNumber(r.size.height),
  ]
}

func integralRectangle(_ r: CGRect) -> Bool {
  let values = [r.origin.x, r.origin.y, r.size.width, r.size.height]
  return values.allSatisfy { $0.isFinite && $0.rounded() == $0 && abs($0) <= 1_000_000 }
    && r.size.width > 0 && r.size.height > 0
}

func parseRectangle(_ text: String) throws -> CGRect {
  let values = text.split(separator: ",", omittingEmptySubsequences: false).compactMap {
    Double($0)
  }
  guard values.count == 4 else {
    throw ProbeError("Crop needs four finite integral point values: x,y,width,height")
  }
  let r = CGRect(x: values[0], y: values[1], width: values[2], height: values[3])
  guard integralRectangle(r) else {
    throw ProbeError("Crop must be finite, positive, nonempty and integral")
  }
  return r
}

// CGWindow/SCWindow use global screen coordinates; sourceRect is display-local
// logical points. This transform never follows a moving window or rounds a crop.
struct CapturePlan: Sendable {
  let displayID: UInt32
  let display: CGRect
  let window: CGRect
  let source: CGRect
  let scale: Double
  let width: Int
  let height: Int

  init(
    displayID: UInt32, display: CGRect, scDisplay: CGRect, window: CGRect, scWindow: CGRect,
    scale: Double, backingScale: Double, requested: CGRect?
  ) throws {
    guard displayID > 0, integralRectangle(display), integralRectangle(window),
      display == scDisplay, window == scWindow, display.contains(window),
      scale.isFinite, (1...4).contains(scale), backingScale == scale
    else {
      throw ProbeError(
        "Window/display identity, integral bounds, containment or independent backing scale disagree"
      )
    }
    let crop = window.offsetBy(dx: -display.origin.x, dy: -display.origin.y)
    guard integralRectangle(crop), CGRect(origin: .zero, size: display.size).contains(crop),
      requested == nil || requested == crop
    else {
      throw ProbeError("Explicit source crop differs from the entire verified owned window")
    }
    let w = window.width * scale
    let h = window.height * scale
    guard w.rounded() == w, h.rounded() == h, w >= 1, h >= 1, w <= 4096, h <= 4096 else {
      throw ProbeError("Native output dimensions are fractional, empty or exceed 4096 pixels")
    }
    self.displayID = displayID
    self.display = display
    self.window = window
    self.source = crop
    self.scale = scale
    self.width = Int(w)
    self.height = Int(h)
  }

  func validateReceipt(
    window: CGRect?, owner: Int32?, expectedPID: Int32, display: CGRect,
    mode: [Int], expectedMode: [Int]
  ) throws {
    guard owner == expectedPID, window == self.window, display == self.display, mode == expectedMode
    else {
      throw ProbeError("Owned target moved, hid, changed owner/display/mode, or became unavailable")
    }
  }

  func validateFrame(
    width: Int, height: Int, content: CGRect?, screen: CGRect?,
    scale: Double?, contentScale: Double?
  ) throws {
    guard width == self.width, height == self.height,
      content == CGRect(origin: .zero, size: window.size), screen == window,
      scale == self.scale, contentScale == 1
    else {
      throw ProbeError(
        "Frame dimensions, crop location, content rectangle or native scales disagree with the frozen plan"
      )
    }
  }

  var json: [String: Any] {
    [
      "display_id": displayID, "display_global_points": rectangle(display),
      "window_global_points": rectangle(window), "source_display_local_points": rectangle(source),
      "point_pixel_scale": scale, "width_pixels": width, "height_pixels": height,
      "crop_is_explicit": true, "crop_follows_target": false,
    ]
  }
}
