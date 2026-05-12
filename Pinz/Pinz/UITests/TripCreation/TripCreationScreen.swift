import Foundation
import XCTest
import PinzAccessibility
import PinzBase

@MainActor
struct TripCreationScreen {
    let app: XCUIApplication

    private let defaultTimeout: TimeInterval = 8

    private var nameField: XCUIElement {
        let identifier = PinzElement.trip(.input(.name)).accessibilityID
        if app.textFields[identifier].exists {
            return app.textFields[identifier]
        }
        return app.otherElements[identifier]
    }

    private var seasonPicker: XCUIElement {
        let identifier = PinzElement.trip(.input(.seasonPicker)).accessibilityID
        let button = app.buttons[identifier]
        if button.exists {
            return button
        }
        return app.descendants(matching: .any).matching(identifier: identifier).firstMatch
    }

    private var categoryPicker: XCUIElement {
        let identifier = PinzElement.trip(.input(.categoryPicker)).accessibilityID
        let button = app.buttons[identifier]
        if button.exists {
            return button
        }
        return app.descendants(matching: .any).matching(identifier: identifier).firstMatch
    }

    private var generatePinsButton: XCUIElement {
        let button = app.buttons["tripCreation.button.generatePins"]
        if button.exists {
            return button
        }
        return app.buttons[PinzBaseStrings.TripCreation.Button.generatePins]
    }

    private var preprocessedNextButton: XCUIElement {
        let button = app.buttons["tripCreation.button.preprocessedNext"]
        if button.exists {
            return button
        }
        return app.buttons[PinzBaseStrings.Common.Button.next]
    }

    private var reviewNextButton: XCUIElement {
        let button = app.buttons["tripCreation.button.reviewNext"]
        if button.exists {
            return button
        }
        return app.buttons[PinzBaseStrings.Common.Button.next]
    }

    @discardableResult
    func waitForInitialSetup(timeout: TimeInterval = 8) -> Bool {
        nameField.waitForExistence(timeout: timeout) && generatePinsButton.waitForExistence(timeout: timeout)
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
    func pickSeason(_ value: String = "Лето") -> Bool {
        pickWheelValue(opening: seasonPicker, value: value)
    }

    @discardableResult
    func pickCategory(_ value: String = "Отпуск") -> Bool {
        pickWheelValue(opening: categoryPicker, value: value)
    }

    @discardableResult
    func tapGeneratePins(timeout: TimeInterval = 8) -> Bool {
        app.dismissKeyboardIfNeeded()
        guard generatePinsButton.waitForExistence(timeout: timeout), generatePinsButton.isEnabled else {
            return false
        }
        generatePinsButton.tap()
        return waitForPreprocessedPins()
    }

    @discardableResult
    func tapPreprocessedNext(timeout: TimeInterval = 8) -> Bool {
        guard preprocessedNextButton.waitForExistence(timeout: timeout), preprocessedNextButton.isEnabled else {
            return false
        }
        preprocessedNextButton.tap()
        return waitForReview()
    }

    @discardableResult
    func tapReviewNext(timeout: TimeInterval = 8) -> Bool {
        guard reviewNextButton.waitForExistence(timeout: timeout), reviewNextButton.isEnabled else {
            return false
        }
        reviewNextButton.tap()
        return true
    }

    @discardableResult
    func waitForPreprocessedPins(timeout: TimeInterval = 10) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if preprocessedNextButton.exists {
                return true
            }
            if app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", PinzBaseStrings.Common.Label.pinNumber(1))).firstMatch.exists {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @discardableResult
    func waitForReview(timeout: TimeInterval = 10) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if app.staticTexts["tripCreation.review.title"].exists {
                return true
            }
            if app.staticTexts[PinzBaseStrings.ReviewTripCreation.Title.main].exists {
                return true
            }
            if app.staticTexts.matching(NSPredicate(format: "label CONTAINS %@", "Created Trip Review Pin")).firstMatch.exists {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    private func pickWheelValue(opening pickerButton: XCUIElement, value: String) -> Bool {
        guard pickerButton.waitForExistence(timeout: defaultTimeout) else {
            return false
        }
        pickerButton.tap()

        let wheel = app.pickerWheels.firstMatch
        guard wheel.waitForExistence(timeout: defaultTimeout) else {
            return false
        }
        wheel.adjust(toPickerWheelValue: value)

        let done = app.buttons[PinzBaseStrings.Common.Button.done]
        guard done.waitForExistence(timeout: defaultTimeout) else {
            return false
        }
        done.tap()
        return true
    }
}
