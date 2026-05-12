import Foundation
import XCTest
import PinzBase

@MainActor
final class TripCreationUITests: XCTestCase {
    private var app: XCUIApplication!
    private var backend: MockBackend!
    private var tripCreationResponseFactory: TripCreationResponseFactory!

    @MainActor
    override func setUp() {
        super.setUp()
        continueAfterFailure = false

        tripCreationResponseFactory = TripCreationResponseFactory()

        do {
            backend = try MockBackend { routes in
                try routes.register(collection: TripCreationController(responseFactory: tripCreationResponseFactory))
            }
        } catch {
            XCTFail("Failed to start trip creation mock backend: \(error)")
            return
        }

        backend.launch()
        XCTAssertTrue(waitForBackendHealth(timeout: 3.0))

        app = XCUIApplication()
        app.launchArguments = [
            PinzLaunchArg.useLocalhost,
            PinzLaunchArg.fakeTokens,
            PinzLaunchArg.testingTripCreation,
            PinzLaunchArg.testingTripCreationFakeMedia
        ]
        app.launch()
    }

    @MainActor
    override func tearDown() {
        app?.terminate()
        backend?.shutdown()
        app = nil
        backend = nil
        tripCreationResponseFactory = nil
        super.tearDown()
    }

    @MainActor
    func test_createTrip_reachesReview() {
        let screen = TripCreationScreen(app: app)

        XCTAssertTrue(screen.waitForInitialSetup())
        screen.setName("Trip_UI_1")
        XCTAssertTrue(screen.tapGeneratePins())
        XCTAssertTrue(screen.tapPreprocessedNext())
    }
}
