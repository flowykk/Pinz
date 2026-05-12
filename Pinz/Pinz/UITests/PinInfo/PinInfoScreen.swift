import Foundation
import XCTest
import PinzAccessibility
import PinzBase

@MainActor
struct PinInfoScreen {
    let app: XCUIApplication

    private let defaultTimeout: TimeInterval = 8

    init(app: XCUIApplication) {
        self.app = app
    }

    private var editButton: XCUIElement {
        app.buttons[PinzElement.pin(.button(.edit)).accessibilityID]
    }

    private var cancelButton: XCUIElement {
        app.buttons[PinzElement.pin(.button(.cancel)).accessibilityID]
    }

    private var doneButton: XCUIElement {
        app.buttons[PinzElement.pin(.button(.done)).accessibilityID]
    }

    private var deleteButton: XCUIElement {
        app.buttons[PinzElement.pin(.button(.delete)).accessibilityID]
    }

    private var headerTitleDetail: XCUIElement {
        app.staticTexts[PinzElement.pin(.row(.headerTitleDetail)).accessibilityID]
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

    private var descriptionField: XCUIElement {
        let identifier = PinzElement.pin(.input(.description)).accessibilityID
        if app.textViews[identifier].exists {
            return app.textViews[identifier]
        }
        if app.textFields[identifier].exists {
            return app.textFields[identifier]
        }
        return app.otherElements[identifier]
    }

    @discardableResult
    func tapEdit(timeout: TimeInterval = 8) -> Bool {
        guard editButton.waitForExistence(timeout: timeout) else {
            return false
        }
        editButton.tap()
        return waitForEditMode()
    }

    @discardableResult
    func tapCancel(timeout: TimeInterval = 6) -> Bool {
        guard cancelButton.waitForExistence(timeout: timeout) else {
            return false
        }
        cancelButton.tap()
        return waitForDefaultMode()
    }

    @discardableResult
    func tapDone(timeout: TimeInterval = 6) -> Bool {
        guard doneButton.waitForExistence(timeout: timeout) else {
            return false
        }
        doneButton.tap()
        return true
    }

    @discardableResult
    func tapDeletePin(timeout: TimeInterval = 6) -> Bool {
        guard deleteButton.waitForExistence(timeout: timeout) else {
            return false
        }
        deleteButton.tap()
        return true
    }

    @discardableResult
    func tapDeletePinConfirm(timeout: TimeInterval = 4) -> Bool {
        let labels = [
            PinzBaseStrings.PinInfo.Alert.DeletePin.confirm,
            "Delete",
            "Удалить"
        ]
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            for label in labels where !label.isEmpty {
                let button = app.buttons[label]
                if button.waitForExistence(timeout: 0.2) {
                    button.tap()
                    return true
                }
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @discardableResult
    func waitForPinInfoToClose(timeout: TimeInterval = 6) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if !editButton.exists && !cancelButton.exists && !doneButton.exists
                && !nameField.exists && !descriptionField.exists
                && !headerTitleDetail.exists {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    func setName(_ value: String) {
        guard nameField.waitForExistence(timeout: defaultTimeout) else {
            return
        }
        nameField.forceTap()
        nameField.clearText()
        nameField.typeText(value)
    }

    func setDescription(_ value: String) {
        guard descriptionField.waitForExistence(timeout: defaultTimeout) else {
            return
        }
        descriptionField.forceTap()
        descriptionField.clearTextCompletely(app: app)
        descriptionField.typeText(value)
    }

    @discardableResult
    func openPinFromPinsList(named name: String, timeout: TimeInterval = 8) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if let pinButton = app.firstHittableOrFirst(
                app.buttons.matching(NSPredicate(format: "label CONTAINS %@", name)),
                timeout: 0.4
            ) {
                pinButton.tap()
                if waitForPinInfoToOpen(timeout: 2.0) {
                    return true
                }
            }

            let textMatches = app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", name))
            if let textElement = app.firstHittableOrFirst(textMatches, timeout: 0.4) {
                textElement.tap()
                if waitForPinInfoToOpen(timeout: 2.0) {
                    return true
                }
            }

            if let fallback = app.firstHittableOrFirst(
                app.descendants(matching: .any).matching(NSPredicate(format: "label CONTAINS %@", name)),
                timeout: 0.4
            ) {
                fallback.tap()
                if waitForPinInfoToOpen(timeout: 2.0) {
                    return true
                }
            }

            Thread.sleep(forTimeInterval: 0.2)
        }
        return false
    }

    @discardableResult
    func waitForPinInfoToOpen(timeout: TimeInterval = 4) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if editButton.exists || headerTitleDetail.exists {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return editButton.exists || headerTitleDetail.exists
    }

    @discardableResult
    func openGallery(timeout: TimeInterval = 6) -> Bool {
        let labels = [
            PinzBaseStrings.Common.Label.gallery,
            "Gallery",
            "Галерея"
        ]
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            for label in labels where !label.isEmpty {
                let button = app.buttons[label]
                if button.exists {
                    button.tap()
                    return waitForGalleryMode(timeout: 3)
                }

                let text = app.staticTexts[label]
                if text.exists {
                    text.tap()
                    return waitForGalleryMode(timeout: 3)
                }
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @discardableResult
    func waitForGalleryMode(timeout: TimeInterval = 4) -> Bool {
        let addMediaIdentifier = PinzElement.pin(.button(.addMedia)).accessibilityID
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if app.buttons[addMediaIdentifier].exists || app.buttons[PinzBaseStrings.PinUpload.Header.addMedia].exists {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @discardableResult
    func waitForDefaultMode(timeout: TimeInterval = 4) -> Bool {
        guard editButton.waitForExistence(timeout: timeout) else {
            return false
        }
        return !cancelButton.exists && !doneButton.exists
    }

    @discardableResult
    func waitForEditMode(timeout: TimeInterval = 4) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            let hasActionButtons = cancelButton.exists || doneButton.exists
            let hasFields = nameField.exists || descriptionField.exists
            if hasActionButtons && hasFields {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @discardableResult
    func waitForToast(_ messages: [String], timeout: TimeInterval = 4) -> Bool {
        let timeoutDate = Date().addingTimeInterval(timeout)
        while Date() < timeoutDate {
            for message in messages {
                let toast = app.staticTexts[message]
                if toast.exists {
                    return true
                }
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @discardableResult
    func waitForToast(_ message: String, timeout: TimeInterval = 4) -> Bool {
        waitForToast([message], timeout: timeout)
    }

    @discardableResult
    func waitForPinDescriptionValue(_ expected: String, timeout: TimeInterval = 6) -> Bool {
        let expectedNormalized = expected.replacingOccurrences(of: "\n", with: " ")
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let predicate = NSPredicate(format: "label CONTAINS %@", expectedNormalized)

        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            let valueElement = app.staticTexts.matching(predicate).firstMatch
            if valueElement.waitForExistence(timeout: 0)
                || valueElement.label.contains(expectedNormalized) {
                return true
            }

            let alt = app.staticTexts.matching(NSPredicate(format: "value CONTAINS %@", expectedNormalized)).firstMatch
            if alt.exists {
                if let value = alt.value as? String, value.contains(expectedNormalized) {
                    return true
                }
            }

            let textView = app.textViews.matching(predicate).firstMatch
            if textView.exists {
                if let value = textView.value as? String, value.contains(expectedNormalized) {
                    return true
                }
                if !textView.label.isEmpty && textView.label.contains(expectedNormalized) {
                    return true
                }
            }

            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }
}
