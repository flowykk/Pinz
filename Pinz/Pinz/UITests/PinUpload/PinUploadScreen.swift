import Foundation
import XCTest
import PinzAccessibility
import PinzBase

@MainActor
struct PinUploadScreen {
    let app: XCUIApplication

    private let defaultTimeout: TimeInterval = 8

    private var addPinButton: XCUIElement {
        let identifier = PinzElement.trip(.button(.addPin)).accessibilityID
        let button = app.buttons[identifier]
        if button.exists {
            return button
        }
        return app.buttons[PinzBaseStrings.TripPins.Button.addPin]
    }

    private var addMediaToPinButton: XCUIElement {
        let identifier = PinzElement.pin(.button(.addMedia)).accessibilityID
        let button = app.buttons[identifier]
        if button.exists {
            return button
        }
        return app.buttons[PinzBaseStrings.PinUpload.Header.addMedia]
    }

    private var nextButton: XCUIElement {
        let button = app.buttons["pinUpload.button.next"]
        if button.exists {
            return button
        }
        return app.buttons[PinzBaseStrings.Common.Button.next]
    }

    private var saveButton: XCUIElement {
        let button = app.buttons["pinUpload.button.save"]
        if button.exists {
            return button
        }
        return app.buttons[PinzBaseStrings.Common.Button.save]
    }

    private var nameField: XCUIElement {
        let identifier = PinzElement.pin(.input(.name)).accessibilityID
        if app.textFields[identifier].exists {
            return app.textFields[identifier]
        }
        if app.textViews[identifier].exists {
            return app.textViews[identifier]
        }
        return app.otherElements[identifier]
    }

    @discardableResult
    func tapAddPin(timeout: TimeInterval = 8) -> Bool {
        guard addPinButton.waitForExistence(timeout: timeout) else {
            return false
        }
        addPinButton.tap()
        return waitForStartScreen()
    }

    @discardableResult
    func tapAddMediaToExistingPin(timeout: TimeInterval = 8) -> Bool {
        guard addMediaToPinButton.waitForExistence(timeout: timeout), addMediaToPinButton.isEnabled else {
            return false
        }
        addMediaToPinButton.tap()
        return waitForStartScreen()
    }

    @discardableResult
    func tapNext(timeout: TimeInterval = 8) -> Bool {
        guard nextButton.waitForExistence(timeout: timeout), nextButton.isEnabled else {
            return false
        }
        nextButton.tap()
        return waitForReviewScreen(timeout: 12)
    }

    func setName(_ value: String) {
        guard nameField.waitForExistence(timeout: defaultTimeout) else {
            return
        }
        nameField.forceTap()
        nameField.clearTextCompletely(app: app)
        nameField.typeText(value)
    }

    @discardableResult
    func tapSave(timeout: TimeInterval = 8) -> Bool {
        app.dismissKeyboardIfNeeded()
        guard saveButton.waitForExistence(timeout: timeout), saveButton.isEnabled else {
            return false
        }
        saveButton.tap()
        return true
    }

    @discardableResult
    func waitForStartScreen(timeout: TimeInterval = 6) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if nextButton.exists {
                return true
            }
            if app.staticTexts[PinzBaseStrings.PinUpload.Header.createPin].exists {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @discardableResult
    func waitForReviewScreen(timeout: TimeInterval = 10) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if nameField.exists && saveButton.exists {
                return true
            }
            if app.staticTexts[PinzBaseStrings.PinUpload.Review.Header.newPin].exists && saveButton.exists {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @discardableResult
    func waitForUploadFlowToClose(timeout: TimeInterval = 6) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if !saveButton.exists && !nameField.exists {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }
}
