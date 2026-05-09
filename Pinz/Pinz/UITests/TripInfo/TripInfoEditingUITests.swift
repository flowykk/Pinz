import Foundation
import XCTest
import PinzBase

@MainActor
final class TripInfoEditingUITests: XCTestCase {
    private var app: XCUIApplication!
    private var backend: MockBackend!
    private var tripResponseFactory: TripInfoResponseFactory!
    private var profileResponseFactory: ProfileResponseFactory!

    private let testingTripId = "trip-ui-001"
    private let baseTripName = "Siberia Route"
    private let baseTripDescription = "Mountain route with lakes and local stories."

    @MainActor
    override func setUp() {
        super.setUp()
        continueAfterFailure = false
        let now = Int(Date().timeIntervalSince1970)
        let shouldFailPatch = name.contains("changeName_Fails")

        tripResponseFactory = TripInfoResponseFactory(
            initialTrip: MockTripInfoSnapshot(
                id: testingTripId,
                name: baseTripName,
                description: baseTripDescription,
                category: "vacation",
                season: "summer",
                privacyLevel: "private",
                coverUrl: "https://cdn.example.com/trips/siberia-01.jpg",
                ownerUserId: "user-123",
                startDateUnix: now - 86_400,
                endDateUnix: now + 86_400,
                createdAtUnix: now
            ),
            patchShouldFail: shouldFailPatch
        )

        profileResponseFactory = ProfileResponseFactory(
            initialProfile: MockProfileSnapshot(
                userId: "user-123",
                username: "Flow",
                nickname: "Flow",
                email: "flow@example.com",
                avatarUrl: nil
            )
        )

        do {
            backend = try MockBackend { routes in
                try routes.register(collection: ProfileController(responseFactory: profileResponseFactory))
                try routes.register(collection: TripInfoController(responseFactory: tripResponseFactory))
            }
        } catch {
            XCTFail("Failed to start trip mock backend: \(error)")
            return
        }

        backend.launch()
        let backendIsHealthy = waitForBackendHealth(timeout: 3.0)
        XCTAssertTrue(backendIsHealthy)

        app = XCUIApplication()
        app.launchArguments = [
            PinzLaunchArg.useLocalhost,
            PinzLaunchArg.fakeTokens,
            PinzLaunchArg.testingProfile,
            PinzLaunchArg.testingTrip,
            PinzLaunchArg.testingTripId,
            testingTripId
        ]
        app.launch()
    }

    @MainActor
    override func tearDown() {
        app?.terminate()
        backend?.shutdown()
        app = nil
        backend = nil
        tripResponseFactory = nil
        profileResponseFactory = nil
        super.tearDown()
    }

    @MainActor
    func test_openTripInfo_loadsData() async throws {
        let screen = TripInfoScreen(app: app)

        XCTAssertTrue(screen.openTrip())
        XCTAssertTrue(screen.waitForTripNameValue(baseTripName))
        XCTAssertTrue(screen.waitForDescriptionValue(baseTripDescription))
        let loadCountReached = await waitForTripGetCount(expected: 1)
        XCTAssertTrue(loadCountReached)
    }

    @MainActor
    func test_editTrip_changeName_Succeeds() async throws {
        let screen = TripInfoScreen(app: app)
        let updatedName = "Siberia Route (2026)"

        XCTAssertTrue(screen.openTrip())
        XCTAssertTrue(screen.tapEdit())
        screen.setName(updatedName)
        XCTAssertTrue(screen.tapDone())

        XCTAssertTrue(screen.waitForDefaultMode())
        XCTAssertTrue(screen.waitForTripNameValue(updatedName))

        let patchCountReached = await waitForPatchTripCount(expected: 1)
        XCTAssertTrue(patchCountReached)
        let patchBody = await tripResponseFactory.lastPatchBody()
        XCTAssertEqual(patchBody?.name, updatedName)
    }

    @MainActor
    func test_editTrip_changeDescription_Succeeds() async throws {
        let screen = TripInfoScreen(app: app)
        let updatedDescription = "Updated mountain route with a cozy lakeside camp and night views."

        XCTAssertTrue(screen.openTrip())
        XCTAssertTrue(screen.tapEdit())
        screen.setDescription(updatedDescription)
        XCTAssertTrue(screen.tapDone())

        XCTAssertTrue(screen.waitForDefaultMode())
        XCTAssertTrue(screen.waitForDescriptionValue(updatedDescription))

        let patchCountReached = await waitForPatchTripCount(expected: 1)
        XCTAssertTrue(patchCountReached)
        let patchBody = await tripResponseFactory.lastPatchBody()
        XCTAssertEqual(patchBody?.description, updatedDescription)
    }

    @MainActor
    func test_editTrip_changeName_Fails() async throws {
        let screen = TripInfoScreen(app: app)
        let updatedName = "Siberia Route (2026)"

        XCTAssertTrue(screen.openTrip())
        XCTAssertTrue(screen.tapEdit())
        screen.setName(updatedName)
        XCTAssertTrue(screen.tapDone())

        let patchCountReached = await waitForPatchTripCount(expected: 1)
        XCTAssertTrue(patchCountReached)

        let patchBody = await tripResponseFactory.lastPatchBody()
        XCTAssertEqual(patchBody?.name, updatedName)
        XCTAssertTrue(screen.waitForEditMode())
        XCTAssertTrue(
            screen.waitForToast([
                "Failed to save trip",
                "Не удалось сохранить путешествие"
            ], timeout: 6)
        )
    }

    @MainActor
    func test_deleteTrip_succeeds() async throws {
        let screen = TripInfoScreen(app: app)

        XCTAssertTrue(screen.openTrip())
        XCTAssertTrue(screen.tapDeleteTrip())
        XCTAssertTrue(screen.tapDeleteTripConfirm())

        let deleteCountReached = await waitForDeleteTripCount(expected: 1)
        XCTAssertTrue(deleteCountReached)

        XCTAssertTrue(
            screen.waitForToast([
                PinzBaseStrings.TripInfo.Toast.tripDeleted,
                "Trip deleted",
                "Путешествие удалено"
            ], timeout: 6)
        )

        XCTAssertTrue(screen.waitForTripInfoToClose())
    }

    @MainActor
    func test_leaveTrip_succeeds() async throws {
        let screen = TripInfoScreen(app: app)

        XCTAssertTrue(screen.openTrip())
        XCTAssertTrue(screen.tapLeaveTrip())
        XCTAssertTrue(screen.tapLeaveTripConfirm())

        let leaveCountReached = await waitForLeaveTripCount(expected: 1)
        XCTAssertTrue(leaveCountReached)

        XCTAssertTrue(
            screen.waitForToast([
                PinzBaseStrings.TripInfo.Toast.tripLeft,
                "You left the trip",
                "Вы покинули путешествие"
            ], timeout: 6)
        )

        XCTAssertTrue(screen.waitForTripInfoToClose())
    }

    private func waitForTripGetCount(expected: Int, timeout: TimeInterval = 2.0) async -> Bool {
        guard let tripResponseFactory else {
            return false
        }

        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if await tripResponseFactory.getTripCount() == expected {
                return true
            }
            try? await Task.sleep(for: .milliseconds(100))
        }
        return false
    }

    private func waitForPatchTripCount(expected: Int, timeout: TimeInterval = 2.0) async -> Bool {
        guard let tripResponseFactory else {
            return false
        }

        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if await tripResponseFactory.patchTripCount() == expected {
                return true
            }
            try? await Task.sleep(for: .milliseconds(100))
        }
        return false
    }

    private func waitForDeleteTripCount(expected: Int, timeout: TimeInterval = 2.0) async -> Bool {
        guard let tripResponseFactory else {
            return false
        }

        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if await tripResponseFactory.deleteTripCount() == expected {
                return true
            }
            try? await Task.sleep(for: .milliseconds(100))
        }
        return false
    }

    private func waitForLeaveTripCount(expected: Int, timeout: TimeInterval = 2.0) async -> Bool {
        guard let tripResponseFactory else {
            return false
        }

        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if await tripResponseFactory.leaveTripCount() == expected {
                return true
            }
            try? await Task.sleep(for: .milliseconds(100))
        }
        return false
    }

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

    private func isBackendHealthy(url: URL) -> Bool {
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.timeoutInterval = 0.25

        let semaphore = DispatchSemaphore(value: 0)
        var isHealthy = false

        let task = URLSession.shared.dataTask(with: request) { _, response, _ in
            defer { semaphore.signal() }
            guard let response = response as? HTTPURLResponse else {
                return
            }
            isHealthy = (200...299).contains(response.statusCode)
        }

        task.resume()
        _ = semaphore.wait(timeout: .now() + 0.3)
        return isHealthy
    }
}
