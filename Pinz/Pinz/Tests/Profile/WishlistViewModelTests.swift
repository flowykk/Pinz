import XCTest
@testable import PinzProfile
import PinzBase
import PinzDomain

@MainActor
final class WishlistViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var sut: WishlistViewModel!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        sut = WishlistViewModel(wishlist: WishlistElement.stubs)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_init_loadsWishlist() {
        XCTAssertEqual(sut.wishlist.count, WishlistElement.stubs.count)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_navigate_wishlistElement_callsRouter() {
        let element = WishlistElement.stubs[0]
        sut.dispatch(.navigate(.wishlistElement(element)))
        XCTAssertNotNil(mockRouter.navigatedWishlistElement)
    }

    func test_navigate_wishlistElementCreation_callsRouterAndAppendsOnCallback() {
        let initialCount = sut.wishlist.count
        sut.dispatch(.navigate(.wishlistElementCreation))
        let action = mockRouter.navigatedWishlistElementCreation
        XCTAssertNotNil(action)
        let newElement = WishlistElement(image: UIImage(), title: "New", description: "Desc")
        action?.action(newElement)
        XCTAssertEqual(sut.wishlist.count, initialCount + 1)
    }
}
