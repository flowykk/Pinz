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
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            guard let factory = wishlistResponseFactory else {
                return 0
            }
            let counts = await factory.getCounts()
            if counts.create == expected {
                return counts.create
            }
            try? await Task.sleep(for: .milliseconds(100))
        }
        guard let factory = wishlistResponseFactory else {
            return 0
        }
        return await factory.getCounts().create
    }

    @MainActor
    private func waitForBackendHealth(timeout: TimeInterval = 2.0) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        let requestURL = URL(string: "http://localhost:8080/health")!

        while Date() < deadline {
            if isBackendHealthy(url: requestURL) {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    @MainActor
    private func isBackendHealthy(url: URL) -> Bool {
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.timeoutInterval = 0.25

        let sema = DispatchSemaphore(value: 0)
        var isHealthy = false

        let task = URLSession.shared.dataTask(with: request) { _, response, _ in
            defer { sema.signal() }
            guard let response = response as? HTTPURLResponse else {
                return
            }
            isHealthy = (200 ... 299).contains(response.statusCode)
        }

        task.resume()
        _ = sema.wait(timeout: .now() + 0.3)
        return isHealthy
    }
}
