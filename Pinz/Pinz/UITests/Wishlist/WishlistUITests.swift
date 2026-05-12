import Foundation
import XCTest
import PinzBase

@MainActor
final class WishlistUITests: XCTestCase {
    private var app: XCUIApplication!
    private var backend: MockBackend!
    private var wishlistResponseFactory: WishlistResponseFactory!
    private var profileResponseFactory: ProfileResponseFactory!

    private let initialWishlist = [
        MockWishlistPlaceSnapshot(
            id: "place-001",
            name: "Old Harbour",
            description: "A cozy place",
            imageUrl: nil
        )
    ]

    @MainActor
    override func setUp() {
        super.setUp()
        continueAfterFailure = false

        profileResponseFactory = ProfileResponseFactory(
            initialProfile: MockProfileSnapshot(
                userId: "user-123",
                username: "Flow",
                nickname: "Flow",
                email: "flow@example.com",
                avatarUrl: nil
            )
        )
        wishlistResponseFactory = WishlistResponseFactory(
            initialWishlist: initialWishlist
        )

        do {
            backend = try MockBackend { routes in
                try routes.register(collection: ProfileController(responseFactory: profileResponseFactory))
                try routes.register(collection: WishlistController(responseFactory: wishlistResponseFactory))
            }
        } catch {
            XCTFail("Failed to start wishlist mock backend: \(error)")
            return
        }

        backend.launch()
        XCTAssertTrue(waitForBackendHealth(timeout: 3.0))

        app = XCUIApplication()
        app.launchArguments = [
            PinzLaunchArg.useLocalhost,
            PinzLaunchArg.fakeTokens,
            PinzLaunchArg.testingProfile,
            PinzLaunchArg.testingWishlist
        ]
        app.launch()
    }

    @MainActor
    override func tearDown() {
        app?.terminate()
        backend?.shutdown()
        app = nil
        backend = nil
        profileResponseFactory = nil
        wishlistResponseFactory = nil
        super.tearDown()
    }

    @MainActor
    func testWishlist_Create_Succeeds() async throws {
        let screen = WishlistScreen(app: app)

        XCTAssertTrue(screen.openProfile())
        XCTAssertTrue(screen.openWishlist())
        XCTAssertTrue(screen.tapAdd())

        screen.setName("Riviera")
        XCTAssertTrue(screen.tapDoneOrNext())
        screen.setDescription("Summer trip place")
        XCTAssertTrue(screen.tapDoneOrNext())
        XCTAssertTrue(screen.tapDoneOrNext())

        XCTAssertTrue(screen.waitForToast(PinzBaseStrings.Wishlist.Toast.placeCreated))

        let createCount = await getCreateCount(expected: 1)
        XCTAssertEqual(createCount, 1)
        XCTAssertTrue(screen.waitForWishlistCell(withId: "place-001"))
        XCTAssertTrue(screen.waitForWishlistCell(withName: "Riviera", timeout: 5))
    }

    @MainActor
    private func getCreateCount(expected: Int, timeout: TimeInterval = 2.0) async -> Int {
        _ = await waitUntil(timeout: timeout) {
            guard let factory = wishlistResponseFactory else {
                return false
            }
            let counts = await factory.getCounts()
            return counts.create == expected
        }
        guard let factory = wishlistResponseFactory else {
            return 0
        }
        return await factory.getCounts().create
    }
}
