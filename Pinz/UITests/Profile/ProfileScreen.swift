import Foundation
import XCTest
import PinzAccessibility

@MainActor
struct ProfileScreen {
    let app: XCUIApplication

    private let defaultTimeout: TimeInterval = 8

    init(app: XCUIApplication) {
        self.app = app
    }

    private var openProfileButton: XCUIElement {
        app.buttons[PinzElement.trip(.button(.openProfile)).accessibilityID]
    }

    private var editButton: XCUIElement {
        app.buttons[PinzElement.profile(.button(.edit)).accessibilityID]
    }

    private var cancelButton: XCUIElement {
        app.buttons[PinzElement.profile(.button(.cancel)).accessibilityID]
    }

    private var backButton: XCUIElement {
        app.buttons[PinzElement.profile(.button(.back)).accessibilityID]
    }

    private var saveButton: XCUIElement {
        let button = app.buttons[PinzElement.profile(.button(.done)).accessibilityID]
        if button.exists {
            return button
        }
        return app.buttons[PinzElement.profile(.button(.save)).accessibilityID]
    }

    private var nicknameField: XCUIElement {
        app.textFields[PinzElement.profile(.input(.nickname)).accessibilityID]
    }

    private var changeEmailButton: XCUIElement {
        app.buttons[PinzElement.settings(.profileEmail).accessibilityID]
    }

    private var emailField: XCUIElement {
        app.textFields[PinzElement.profile(.input(.email)).accessibilityID]
    }

    private var headerNickname: XCUIElement {
        app.staticTexts[PinzElement.profile(.row(.headerNickname)).accessibilityID]
    }

    private var profileEmailSetting: XCUIElement {
        app.buttons[PinzElement.settings(.profileEmail).accessibilityID]
    }

    @discardableResult
    func changeEmailField(index: Int) -> XCUIElement {
        let identifier = PinzElement.profile(.input(.verificationCode(index))).accessibilityID
        let textField = app.textFields[identifier]
        if textField.exists {
            return textField
        }
        return app.secureTextFields[identifier]
    }

    @discardableResult
    func waitForMainProfileButton(timeout: TimeInterval = 8) -> Bool {
        openProfileButton.waitForExistence(timeout: timeout)
    }

    @discardableResult
    func openProfile() -> Bool {
        guard waitForMainProfileButton(timeout: defaultTimeout) else {
            return false
        }
        openProfileButton.tap()
        return editButton.waitForExistence(timeout: defaultTimeout)
    }

    @discardableResult
    func tapEdit() -> Bool {
        guard editButton.waitForExistence(timeout: defaultTimeout) else {
            return false
        }
        editButton.tap()
        return nicknameField.waitForExistence(timeout: defaultTimeout)
    }

    @discardableResult
    func tapChangeEmail() -> Bool {
        guard changeEmailButton.waitForExistence(timeout: defaultTimeout) else {
            return false
        }
        changeEmailButton.tap()
        return emailField.waitForExistence(timeout: defaultTimeout)
    }

    @discardableResult
    func tapCancel() -> Bool {
        guard cancelButton.waitForExistence(timeout: defaultTimeout) else {
            return false
        }
        cancelButton.tap()
        return editButton.waitForExistence(timeout: defaultTimeout)
    }

    func setNickname(_ value: String) {
        guard nicknameField.waitForExistence(timeout: defaultTimeout) else {
            return
        }
        nicknameField.tap()

        if let current = nicknameField.value as? String {
            let backspaces = String(repeating: "\u{8}", count: current.count)
            if !backspaces.isEmpty {
                nicknameField.typeText(backspaces)
            }
        }

        nicknameField.typeText(value)
    }

    func setEmail(_ value: String) {
        guard emailField.waitForExistence(timeout: defaultTimeout) else {
            return
        }
        emailField.tap()

        if let current = emailField.value as? String {
            let backspaces = String(repeating: "\u{8}", count: current.count)
            if !backspaces.isEmpty {
                emailField.typeText(backspaces)
            }
        }

        emailField.typeText(value)
    }

    @discardableResult
    func tapDone() -> Bool {
        guard saveButton.waitForExistence(timeout: defaultTimeout) else {
            return false
        }
        saveButton.tap()
        return true
    }

    @discardableResult
    func tapReceiveCode() -> Bool {
        return tapDone()
    }

    @discardableResult
    func tapCodeConfirm() -> Bool {
        return tapDone()
    }

    @discardableResult
    func tapEmailChangeBack() -> Bool {
        guard backButton.waitForExistence(timeout: defaultTimeout) else {
            return false
        }
        backButton.tap()
        return isInEditMode()
    }

    @discardableResult
    func waitForVerificationCode(timeout: TimeInterval = 4.0) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if changeEmailField(index: 0).waitForExistence(timeout: 0.2) {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @discardableResult
    func emailFieldExists(timeout: TimeInterval = 2.0) -> Bool {
        return emailField.waitForExistence(timeout: timeout)
    }

    func enterEmailCode(_ code: String) {
        let characters = Array(code)
        for (index, character) in characters.enumerated() {
            let field = changeEmailField(index: index)
            field.tap()
            field.typeText(String(character))
            Thread.sleep(forTimeInterval: 0.1)
        }
    }

    @discardableResult
    func tapSave() -> Bool {
        guard saveButton.waitForExistence(timeout: defaultTimeout) else {
            return false
        }
        saveButton.tap()
        return true
    }

    @discardableResult
    func waitForHeaderNickname(_ expectedNickname: String, timeout: TimeInterval = 8) -> Bool {
        return waitForHeaderValue(expectedNickname, timeout: timeout)
    }

    @discardableResult
    func waitForHeaderEmail(_ expectedEmail: String, timeout: TimeInterval = 8) -> Bool {
        let expectedWithBullet = "• \(expectedEmail)"
        let expectedWithPipe = "\(expectedEmail) •"
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if headerContains(expectedWithBullet, fallbackWithPipe: expectedWithPipe) {
                return true
            }
            if waitForProfileEmailInSettings(expectedEmail, timeout: 0.2) {
                return true
            }
            Thread.sleep(forTimeInterval: 0.2)
        }

        return waitForProfileEmailInSettings(expectedEmail, timeout: 0.2)
    }

    @discardableResult
    private func waitForProfileEmailInSettings(_ expectedEmail: String, timeout: TimeInterval = 4.0) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if profileEmailSetting.waitForExistence(timeout: 0.2) {
                let exactMatch = profileEmailSetting
                    .staticTexts
                    .matching(NSPredicate(format: "label == %@", expectedEmail))
                    .firstMatch
                if exactMatch.exists {
                    return true
                }

                let containsMatch = profileEmailSetting
                    .staticTexts
                    .matching(NSPredicate(format: "label CONTAINS %@", expectedEmail))
                    .firstMatch
                if containsMatch.exists {
                    return true
                }
            }

            Thread.sleep(forTimeInterval: 0.2)
        }
        return false
    }

    private func waitForHeaderValue(_ expected: String, timeout: TimeInterval) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            let headerText = headerText(for: headerNickname)
            let headerValueText = headerValue(for: headerNickname)

            if headerText == expected || headerValueText == expected || headerText.contains(expected) || headerValueText.contains(expected) {
                return true
            }
            if headerContains(expected) {
                return true
            }
            Thread.sleep(forTimeInterval: 0.2)
        }
        return false
    }

    private func headerContains(_ expected: String, fallbackWithPipe: String? = nil) -> Bool {
        let texts = [headerValue(for: headerNickname), headerText(for: headerNickname)]
        if texts.contains(where: { $0 == expected }) {
            return true
        }

        if headerValue(for: headerNickname).contains(expected) {
            return true
        }

        if headerText(for: headerNickname).contains(expected) {
            return true
        }

        if let fallbackWithPipe, headerText(for: headerNickname).contains(fallbackWithPipe) {
            return true
        }
        return false
    }

    private func headerText(for element: XCUIElement) -> String {
        if element.exists {
            return element.label
        }
        return ""
    }

    private func headerValue(for element: XCUIElement) -> String {
        guard element.exists, let raw = element.value as? String else {
            return ""
        }
        return raw
    }

    @discardableResult
    func waitForValidationToast(_ message: String, timeout: TimeInterval = 4) -> Bool {
        return waitForValidationToast([message], timeout: timeout)
    }

    @discardableResult
    func waitForValidationToast(_ messages: [String], timeout: TimeInterval = 4) -> Bool {
        let timeoutDate = Date().addingTimeInterval(timeout)
        while Date() < timeoutDate {
            for message in messages {
                if app.staticTexts[message].exists {
                    return true
                }

                let toastCandidate = app
                    .staticTexts
                    .matching(NSPredicate(format: "label == %@", message))
                    .firstMatch

                if toastCandidate.exists {
                    return true
                }
            }

            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @discardableResult
    func isInEditMode(timeout: TimeInterval = 4) -> Bool {
        guard cancelButton.waitForExistence(timeout: timeout) else {
            return false
        }
        return nicknameField.waitForExistence(timeout: timeout)
    }
}
