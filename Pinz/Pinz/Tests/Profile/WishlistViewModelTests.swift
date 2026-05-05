import XCTest
@testable import PinzProfile
import PinzBase
import PinzDomain

@MainActor
final class WishlistViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockService: MockNetworkService!
    private var sut: WishlistViewModel!

    private let stubPlaces: [DesiredPlace] = [
        DesiredPlace(id: "1", name: "Мачу-Пикчу", description: "Древний город в горах Перу."),
        DesiredPlace(id: "2", name: "Киото", description: "Японские сады и сакура."),
    ]

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockService = MockNetworkService()
        sut = WishlistViewModel(wishlist: stubPlaces, networkService: mockService)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    func test_init_loadsWishlist() {
        XCTAssertEqual(sut.wishlist.count, stubPlaces.count)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_navigate_wishlistElement_callsRouter() {
        let element = stubPlaces[0]
        sut.dispatch(.navigate(.wishlistElement(element)))
        XCTAssertNotNil(mockRouter.navigatedWishlistElement)
    }

    func test_navigate_wishlistElementCreation_callsRouterAndAppendsOnCallback() {
        let initialCount = sut.wishlist.count
        sut.dispatch(.navigate(.wishlistElementCreation))
        let action = mockRouter.navigatedWishlistElementCreation
        XCTAssertNotNil(action)
        let newElement = DesiredPlace(id: "99", name: "New", description: "Desc")
        action?.action(newElement)
        XCTAssertEqual(sut.wishlist.count, initialCount + 1)
    }

    func test_loadWishlist_populatesWishlist() async {
        let dto1 = DesiredPlaceDTO(id: "dp-1", name: "Токио", description: "Мечта", imageUrl: nil, createdAt: 0)
        let dto2 = DesiredPlaceDTO(id: "dp-2", name: "Рим", description: "Вечный город", imageUrl: nil, createdAt: 0)
        mockService.getDesiredPlacesResult = .success([dto1, dto2])
        sut = WishlistViewModel(networkService: mockService)

        await sut.loadWishlist()

        XCTAssertEqual(sut.wishlist.count, 2)
        XCTAssertEqual(sut.wishlist[0].name, "Токио")
        XCTAssertEqual(sut.wishlist[1].name, "Рим")
    }

    func test_loadWishlist_onNetworkError_keepsPreviousWishlist() async {
        mockService.getDesiredPlacesResult = .failure(URLError(.notConnectedToInternet))

        await sut.loadWishlist()

        XCTAssertEqual(sut.wishlist.count, stubPlaces.count)
    }

    func test_loadWishlist_onNetworkError_showsToast() async {
        var toasts: [String] = []
        sut.setToast { toasts.append($0) }
        mockService.getDesiredPlacesResult = .failure(URLError(.notConnectedToInternet))

        await sut.loadWishlist()

        XCTAssertEqual(toasts, [PinzBaseStrings.Wishlist.Toast.loadFailed])
    }
}
