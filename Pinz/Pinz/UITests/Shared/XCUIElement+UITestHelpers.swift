import Foundation
import XCTest

@MainActor
extension XCUIElement {
    func forceTap() {
        if isHittable {
            tap()
            return
        }

        coordinate(withNormalizedOffset: CGVector(dx: 0.5, dy: 0.5)).tap()
    }

    func clearText() {
        guard let currentValue = value as? String else {
            return
        }

        let backspaces = String(repeating: "\u{8}", count: max(0, currentValue.count))
        if !backspaces.isEmpty {
            typeText(backspaces)
        }
    }

    func clearTextCompletely(app: XCUIApplication? = nil) {
        if let app {
            press(forDuration: 1.2)
            let selectAllMenu = app.menuItems["Select All"]
            if selectAllMenu.waitForExistence(timeout: 0.6) {
                selectAllMenu.tap()
                typeText("\u{8}")
                return
            }
        }

        typeText(XCUIKeyboardKey.command.rawValue + "a")
        typeText("\u{8}")
        clearText()
    }
}

@MainActor
extension XCUIApplication {
    func dismissKeyboardIfNeeded() {
        if keyboards.count > 0 {
            typeText("\n")
        }
    }

    func firstHittableOrFirst(_ query: XCUIElementQuery, timeout: TimeInterval) -> XCUIElement? {
        guard query.count > 0 else { return nil }

        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            for index in 0..<query.count {
                let candidate = query.element(boundBy: index)
                if candidate.exists && candidate.isHittable {
                    return candidate
                }
            }
            Thread.sleep(forTimeInterval: 0.1)
        }

        let first = query.firstMatch
        return first.exists ? first : nil
    }
}
