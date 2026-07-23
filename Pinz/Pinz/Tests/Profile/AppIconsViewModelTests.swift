import XCTest
@testable import PinzProfile

@MainActor
final class AppIconsViewModelTests: XCTestCase {

    private var sut: AppIconsViewModel!

    override func setUp() {
        super.setUp()
        UserDefaults.standard.removeObject(forKey: AppIconViewModel.appIconKey)
        sut = AppIconsViewModel()
    }

    override func tearDown() {
        UserDefaults.standard.removeObject(forKey: AppIconViewModel.appIconKey)
        sut = nil
        super.tearDown()
    }

    func test_init_loadsAllIcons() {
        XCTAssertEqual(sut.appIcons.count, 6)
    }

    func test_init_firstIconIsDefault() {
        XCTAssertEqual(sut.appIcons.first?.name, AppIconViewModel.defaultAppIcon)
    }

    func test_init_exactlyOneIconSelected() {
        let selected = sut.appIcons.filter { $0.selected }
        XCTAssertEqual(selected.count, 1)
    }
}
