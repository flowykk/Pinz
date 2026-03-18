import XCTest
@testable import PinzProfile
import PinzBase
import PinzDomain

final class ProfileViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: ProfileViewModel!

    private let testUser = User(nickname: "tester", email: "test@example.com")

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = ProfileViewModel(user: testUser)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_initialState() {
        XCTAssertEqual(sut.state, .default)
        XCTAssertEqual(sut.user.nickname, "tester")
    }

    func test_changeState_togglesFromDefaultToEditing() {
        sut.dispatch(.changeState)
        XCTAssertEqual(sut.state, .editing)
    }

    func test_changeState_togglesFromEditingToDefault() {
        sut.dispatch(.changeState)
        sut.dispatch(.changeState)
        XCTAssertEqual(sut.state, .default)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_navigate_statistics_callsRouter() {
        sut.dispatch(.navigate(.statistics))
        XCTAssertTrue(mockRouter.navigatedToStatistics)
    }

    func test_navigate_trips_callsRouter() {
        sut.dispatch(.navigate(.trips))
        XCTAssertTrue(mockRouter.navigatedToTrips)
    }

    func test_navigate_wishlist_callsRouter() {
        sut.dispatch(.navigate(.wishlist))
        XCTAssertTrue(mockRouter.navigatedToPlacesWishlist)
    }

    func test_navigate_saved_callsRouter() {
        sut.dispatch(.navigate(.saved))
        XCTAssertTrue(mockRouter.navigatedToSavedMaps)
    }

    func test_navigate_notifications_callsRouter() {
        sut.dispatch(.navigate(.notifications))
        XCTAssertTrue(mockRouter.navigatedToNotifications)
    }

    func test_navigate_appearance_callsRouter() {
        sut.dispatch(.navigate(.appearance))
        XCTAssertTrue(mockRouter.navigatedToAppearance)
    }

    func test_navigate_emailChange_callsRouterWithEmail() {
        sut.dispatch(.navigate(.emailChange))
        XCTAssertEqual(mockRouter.navigatedEmailChange?.email, testUser.email)
    }
}
