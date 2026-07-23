import Foundation
import XCTest
import PinzAccessibility

@MainActor
struct WishlistScreen {
    let app: XCUIApplication

    private let defaultTimeout: TimeInterval = 8

    init(app: XCUIApplication) {
        self.app = app
    }

    private var openProfileButton: XCUIElement {
        app.buttons[PinzElement.trip(.button(.openProfile)).accessibilityID]
    }

    private var openWishlistButton: XCUIElement {
        app.buttons[PinzElement.settings(.profileWishlist).accessibilityID]
    }

    private var addWishlistButton: XCUIElement {
        app.buttons[PinzElement.wishlist(.button(.add)).accessibilityID]
    }

    private var wishlistNameField: XCUIElement {
        app.textFields[PinzElement.wishlist(.input(.name)).accessibilityID]
    }

    private var wishlistCreateDoneButton: XCUIElement {
        app.buttons[PinzElement.wishlist(.button(.done)).accessibilityID]
    }

    @discardableResult
    func openProfile() -> Bool {
        guard openProfileButton.waitForExistence(timeout: defaultTimeout) else {
            return false
        }
        openProfileButton.tap()
        return true
    }

    @discardableResult
    func openWishlist() -> Bool {
        guard openWishlistButton.waitForExistence(timeout: defaultTimeout) else {
            return false
        }
        openWishlistButton.tap()
        return addWishlistButton.waitForExistence(timeout: defaultTimeout)
    }

    @discardableResult
    func tapAdd() -> Bool {
        guard addWishlistButton.waitForExistence(timeout: defaultTimeout) else { return false }
        addWishlistButton.tap()
        return wishlistNameField.waitForExistence(timeout: defaultTimeout)
    }

    func setName(_ value: String) {
        guard let nameField = wishlistNameInput() else { return }
        nameField.forceTap()
        nameField.clearText()
        nameField.typeText(value)
    }

    func setDescription(_ value: String) {
        guard let descriptionField = wishlistDescriptionInput() else { return }
        descriptionField.forceTap()
        descriptionField.clearText()
        descriptionField.typeText(value)
    }

    @discardableResult
    func tapDoneOrNext() -> Bool {
        guard wishlistCreateDoneButton.waitForExistence(timeout: defaultTimeout) else { return false }
        wishlistCreateDoneButton.tap()
        return true
    }

    @discardableResult
    func waitForWishlistCell(withName expectedName: String, timeout: TimeInterval = 6) -> Bool {
        let predicate = NSPredicate(format: "label CONTAINS %@", expectedName)
        let item = app.descendants(matching: .any).matching(predicate).firstMatch

        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if item.waitForExistence(timeout: 0.3) {
                return true
            }
            Thread.sleep(forTimeInterval: 0.2)
        }
        return item.exists
    }

    @discardableResult
    func waitForWishlistCell(withId id: String, timeout: TimeInterval = 6) -> Bool {
        guard let item = wishlistItemElement(withId: id, timeout: timeout) else {
            return false
        }
        return item.waitForExistence(timeout: timeout)
    }

    @discardableResult
    func waitForToast(_ message: String, timeout: TimeInterval = 4) -> Bool {
        let timeoutDate = Date().addingTimeInterval(timeout)
        while Date() < timeoutDate {
            let toast = app.staticTexts[message]
            if toast.exists { return true }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    private func wishlistItemElement(withId id: String, timeout: TimeInterval = 1.0) -> XCUIElement? {
        let identifier = PinzElement.wishlist(.row(.item(id))).accessibilityID

        if let button = app.firstHittableOrFirst(app.buttons.matching(identifier: identifier), timeout: timeout) {
            return button
        }
        if let cell = app.firstHittableOrFirst(app.cells.matching(identifier: identifier), timeout: timeout) {
            return cell
        }
        if let other = app.firstHittableOrFirst(app.otherElements.matching(identifier: identifier), timeout: timeout) {
            return other
        }
        if let image = app.firstHittableOrFirst(app.images.matching(identifier: identifier), timeout: timeout) {
            return image
        }
        let anyMatch = app.descendants(matching: .any).matching(identifier: identifier).firstMatch
        if anyMatch.waitForExistence(timeout: timeout) {
            return anyMatch
        }
        return nil
    }

    private func wishlistNameInput() -> XCUIElement? {
        let namedById = app.textFields[PinzElement.wishlist(.input(.name)).accessibilityID]
        if namedById.exists {
            return namedById
        }

        let field = app.textViews[PinzElement.wishlist(.input(.name)).accessibilityID]
        if field.exists {
            return field
        }

        let fallback = app.otherElements[PinzElement.wishlist(.input(.name)).accessibilityID]
        if fallback.exists {
            return fallback
        }

        return nil
    }

    private func wishlistDescriptionInput() -> XCUIElement? {
        let namedById = app.textFields[PinzElement.wishlist(.input(.description)).accessibilityID]
        if namedById.exists {
            return namedById
        }

        let field = app.textViews[PinzElement.wishlist(.input(.description)).accessibilityID]
        if field.exists {
            return field
        }

        let fallback = app.otherElements[PinzElement.wishlist(.input(.description)).accessibilityID]
        if fallback.exists {
            return fallback
        }

        return nil
    }
}
