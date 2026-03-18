import XCTest
@testable import PinzProfile

@MainActor
final class AppIconViewModelTests: XCTestCase {

    private let testKey = AppIconViewModel.appIconKey
    private let defaultIcon = AppIconViewModel.defaultAppIcon

    override func setUp() {
        super.setUp()
        UserDefaults.standard.removeObject(forKey: testKey)
    }

    override func tearDown() {
        UserDefaults.standard.removeObject(forKey: testKey)
        super.tearDown()
    }

    func test_init_defaultIcon_isSelectedWhenNoSavedIcon() {
        let sut = AppIconViewModel(appIcon: defaultIcon)
        XCTAssertTrue(sut.selected)
    }

    func test_init_nonDefaultIcon_isNotSelectedWhenNoSavedIcon() {
        let sut = AppIconViewModel(appIcon: "PinzLight")
        XCTAssertFalse(sut.selected)
    }

    func test_init_selectedWhenMatchesSavedIcon() {
        UserDefaults.standard.set("PinzLight", forKey: testKey)
        let sut = AppIconViewModel(appIcon: "PinzLight")
        XCTAssertTrue(sut.selected)
    }

    func test_init_notSelectedWhenDoesNotMatchSavedIcon() {
        UserDefaults.standard.set("PinzLight", forKey: testKey)
        let sut = AppIconViewModel(appIcon: defaultIcon)
        XCTAssertFalse(sut.selected)
    }

    func test_iconName_isSetCorrectly() {
        let sut = AppIconViewModel(appIcon: "PinzPin")
        XCTAssertEqual(sut.name, "PinzPin")
    }
}
