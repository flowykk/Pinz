import Foundation
import XCTest
import PinzAccessibility
import PinzBase

@MainActor
struct TripInfoScreen {
    let app: XCUIApplication

    private let defaultTimeout: TimeInterval = 8

    init(app: XCUIApplication) {
        self.app = app
    }

    private var editButton: XCUIElement {
        app.buttons[PinzElement.trip(.button(.edit)).accessibilityID]
    }

    private var cancelButton: XCUIElement {
        app.buttons[PinzElement.trip(.button(.cancel)).accessibilityID]
    }

    private var doneButton: XCUIElement {
        app.buttons[PinzElement.trip(.button(.done)).accessibilityID]
    }

    private var leaveButton: XCUIElement {
        app.buttons[PinzElement.trip(.button(.leave)).accessibilityID]
    }

    private var deleteButton: XCUIElement {
        app.buttons[PinzElement.trip(.button(.delete)).accessibilityID]
    }

    private var nameField: XCUIElement {
        let identifier = PinzElement.trip(.input(.name)).accessibilityID
        let directField = app.textFields[identifier]
        if directField.exists {
            return directField
        }

        let textArea = app.textViews[identifier]
        if textArea.exists {
            return textArea
        }

        return app.otherElements[identifier]
    }

    private var startDatePicker: XCUIElement {
        app.buttons[PinzElement.trip(.input(.startDatePicker)).accessibilityID]
    }

    private var endDatePicker: XCUIElement {
        app.buttons[PinzElement.trip(.input(.endDatePicker)).accessibilityID]
    }

    private var descriptionField: XCUIElement {
        let identifier = PinzElement.trip(.input(.description)).accessibilityID
        let directTextView = app.textViews[identifier]
        if directTextView.exists {
            return directTextView
        }

        let directField = app.textFields[identifier]
        if directField.exists {
            return directField
        }

        return app.otherElements[identifier]
    }

    private var tripNameStaticTextElements: XCUIElementQuery {
        app.staticTexts.matching(identifier: PinzElement.trip(.row(.headerTitleDetail)).accessibilityID)
    }

    private var tripDescriptionStaticTextElements: XCUIElementQuery {
        app.staticTexts.matching(identifier: PinzElement.trip(.row(.description)).accessibilityID)
    }

    private var pinsButton: XCUIElement {
        let identifier = PinzElement.trip(.row(.pins)).accessibilityID
        let button = app.buttons[identifier]
        if button.exists { return button }
        let other = app.otherElements[identifier]
        if other.exists { return other }
        return app.descendants(matching: .any).matching(identifier: identifier).firstMatch
    }

    @discardableResult
    func openTrip(timeout: TimeInterval = 8) -> Bool {
        let openButton = app.buttons[PinzElement.trip(.button(.openTripInfo)).accessibilityID]
        guard openButton.waitForExistence(timeout: timeout) else {
            return false
        }
        openButton.tap()
        return waitForDefaultMode()
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
    func tapLeaveTrip(timeout: TimeInterval = 6) -> Bool {
        guard leaveButton.waitForExistence(timeout: timeout) else {
            return false
        }
        leaveButton.tap()
        return true
    }

    @discardableResult
    func tapDeleteTrip(timeout: TimeInterval = 6) -> Bool {
        guard deleteButton.waitForExistence(timeout: timeout) else {
            return false
        }
        deleteButton.tap()
        return true
    }

    @discardableResult
    func tapLeaveTripConfirm(timeout: TimeInterval = 4) -> Bool {
        let labels = [
            PinzBaseStrings.TripInfo.Alert.LeaveTrip.confirm,
            "Leave",
            "Выйти"
        ]
        return tapDialogButton(labels, timeout: timeout)
    }

    @discardableResult
    func tapDeleteTripConfirm(timeout: TimeInterval = 4) -> Bool {
        let labels = [
            PinzBaseStrings.TripInfo.Alert.DeleteTrip.confirm,
            "Delete",
            "Удалить"
        ]
        return tapDialogButton(labels, timeout: timeout)
    }

    @discardableResult
    func tapPins(timeout: TimeInterval = 6) -> Bool {
        guard pinsButton.waitForExistence(timeout: timeout) else {
            return false
        }
        pinsButton.tap()
        return true
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

    func pickDates(start: Date, end: Date, dateSettleTimeout: TimeInterval = 2) -> Bool {
        if !pickDate(startDate: start, pickerButton: startDatePicker) {
            return false
        }
        Thread.sleep(forTimeInterval: dateSettleTimeout)
        return pickDate(startDate: end, pickerButton: endDatePicker)
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
        guard cancelButton.waitForExistence(timeout: timeout) else {
            return false
        }
        return nameField.waitForExistence(timeout: timeout)
    }

    @discardableResult
    func waitForTripInfoToClose(timeout: TimeInterval = 6) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if !editButton.exists && !cancelButton.exists && !doneButton.exists
                && !nameField.exists && !descriptionField.exists
                && !leaveButton.exists && !deleteButton.exists {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @discardableResult
    func waitForTripNameValue(_ expected: String, timeout: TimeInterval = 6) -> Bool {
        let timeoutDate = Date().addingTimeInterval(timeout)

        while Date() < timeoutDate {
            if editButton.exists && waitForEditMode(timeout: 0) {
                if let fieldValue = accessibleTextValue(nameField), fieldValue == expected {
                    return true
                }
            }

            if let headerValue = tripNameTextValue(contains: expected), !headerValue.isEmpty {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @discardableResult
    func waitForDescriptionValue(_ expected: String, timeout: TimeInterval = 6) -> Bool {
        let timeoutDate = Date().addingTimeInterval(timeout)

        while Date() < timeoutDate {
            if editButton.exists && waitForEditMode(timeout: 0) {
                if let fieldValue = accessibleTextValue(descriptionField), fieldValue == expected {
                    return true
                }
            }

            if let descriptionValue = tripDescriptionTextValue(exact: expected), !descriptionValue.isEmpty {
                return true
            }

            if let descriptionValue = tripDescriptionTextValue(contains: expected), !descriptionValue.isEmpty {
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

    private func tripNameTextValue(exact expected: String) -> String? {
        return textValue(from: tripNameStaticTextElements, exact: expected)
    }

    private func tripNameTextValue(contains expected: String) -> String? {
        return textValue(from: tripNameStaticTextElements, contains: expected)
    }

    private func tripDescriptionTextValue(exact expected: String) -> String? {
        return textValue(from: tripDescriptionStaticTextElements, exact: expected)
    }

    private func tripDescriptionTextValue(contains expected: String) -> String? {
        return textValue(from: tripDescriptionStaticTextElements, contains: expected)
    }

    private func textValue(from query: XCUIElementQuery, exact expected: String) -> String? {
        let exactMatch = query.matching(NSPredicate(format: "label == %@", expected)).firstMatch
        if exactMatch.exists, let value = accessibleTextValue(exactMatch), value == expected {
            return value
        }
        let valueMatch = query.matching(NSPredicate(format: "value == %@", expected)).firstMatch
        if valueMatch.exists, let value = accessibleTextValue(valueMatch), value == expected {
            return value
        }
        return nil
    }

    private func textValue(from query: XCUIElementQuery, contains expected: String) -> String? {
        let containsMatch = query.matching(NSPredicate(format: "label CONTAINS %@", expected)).firstMatch
        if containsMatch.exists, let value = accessibleTextValue(containsMatch), value.contains(expected) {
            return value
        }
        let valueContainsMatch = query.matching(NSPredicate(format: "value CONTAINS %@", expected)).firstMatch
        if valueContainsMatch.exists, let value = accessibleTextValue(valueContainsMatch), value.contains(expected) {
            return value
        }
        return nil
    }

    @discardableResult
    func waitForToast(_ message: String, timeout: TimeInterval = 4) -> Bool {
        waitForToast([message], timeout: timeout)
    }

    private func pickDate(startDate: Date, pickerButton: XCUIElement) -> Bool {
        guard pickerButton.waitForExistence(timeout: defaultTimeout) else {
            return false
        }
        pickerButton.tap()

        let doneButton = app.buttons[PinzBaseStrings.Common.Button.done]
        guard doneButton.waitForExistence(timeout: 1.0) else {
            return false
        }

        let wheels = app.pickerWheels
        guard wheels.count > 0 else {
            doneButton.tap()
            return true
        }

        let calendar = Calendar.current
        let day = calendar.component(.day, from: startDate)
        let month = DateFormatter.monthLocalizedName(from: startDate)
        let year = "\(calendar.component(.year, from: startDate))"

        for idx in 0..<wheels.count {
            let wheel = wheels.element(boundBy: idx)
            if idx == 0 {
                wheel.adjust(toPickerWheelValue: year)
            } else if idx == 1 {
                wheel.adjust(toPickerWheelValue: month)
            } else if idx == 2 {
                wheel.adjust(toPickerWheelValue: "\(day)")
            }
        }

        doneButton.tap()
        return true
    }

    private func tapDialogButton(_ labels: [String], timeout: TimeInterval) -> Bool {
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

    private func accessibleTextValue(_ element: XCUIElement) -> String? {
        guard element.exists else {
            return nil
        }

        let normalizedLabel = element.label
            .replacingOccurrences(of: "\n", with: " ")
            .trimmingCharacters(in: .whitespacesAndNewlines)

        if let value = element.value as? String {
            let normalizedValue = value
                .replacingOccurrences(of: "\n", with: " ")
                .trimmingCharacters(in: .whitespacesAndNewlines)
            if !normalizedValue.isEmpty {
                return normalizedValue
            }
        }

        return normalizedLabel.isEmpty ? nil : normalizedLabel
    }
}

private extension DateFormatter {
    static func monthLocalizedName(from date: Date) -> String {
        let formatter = DateFormatter()
        formatter.calendar = Calendar.current
        formatter.locale = Locale.current
        formatter.dateFormat = "LLLL"
        return formatter.string(from: date)
    }
}
