import Foundation
import XCTest
import PinzBase
import PinzDomain

@MainActor
final class PinInfoEditingUITests: XCTestCase {
    private var app: XCUIApplication!
    private var backend: MockBackend!
    private var tripResponseFactory: TripInfoResponseFactory!
    private var profileResponseFactory: ProfileResponseFactory!

    private let testingTripId = "trip-ui-pin-001"
    private let baseTripName = "UI Trip (Pins)"
    private let baseTripDescription = "Trip created for PinInfo UI tests."

    private let testingPinId = "pin-ui-001"
    private let basePinName = "Old Pin Name"
    private let basePinDescription = "Base pin description."

    @MainActor
    override func setUp() {
        super.setUp()
        continueAfterFailure = false

        let now = Int(Date().timeIntervalSince1970)
        let shouldFailPinPatch = name.contains("changeName_Fails")

        let initialPin = TripPinDTO(
            id: testingPinId,
            tripId: testingTripId,
            name: basePinName,
            description: basePinDescription,
            category: "sight",
            latitude: 55.7558,
            longitude: 37.6176,
            startTimeUnix: nil,
            endTimeUnix: nil,
            tags: [],
            privacyLevel: "private",
            media: []
        )

        tripResponseFactory = TripInfoResponseFactory(
            initialTrip: MockTripInfoSnapshot(
                id: testingTripId,
                name: baseTripName,
                description: baseTripDescription,
                category: "vacation",
                season: "summer",
                privacyLevel: "private",
                coverUrl: nil,
                ownerUserId: "user-123",
                startDateUnix: now - 86_400,
                endDateUnix: now + 86_400,
                createdAtUnix: now,
                initialPins: [initialPin]
            ),
            patchPinShouldFail: shouldFailPinPatch
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
            XCTFail("Failed to start pin info mock backend: \(error)")
            return
        }

        backend.launch()
        XCTAssertTrue(waitForBackendHealth(timeout: 3.0))

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
    func test_editPin_changeName_Succeeds() async throws {
        let tripInfo = TripInfoScreen(app: app)
        let pinInfo = PinInfoScreen(app: app)

        XCTAssertTrue(tripInfo.openTrip())
        XCTAssertTrue(tripInfo.tapPins())
        XCTAssertTrue(pinInfo.openPinFromPinsList(named: basePinName))

        XCTAssertTrue(pinInfo.tapEdit())

        let updatedName = "Updated Pin Name"
        pinInfo.setName(updatedName)
        XCTAssertTrue(pinInfo.tapDone())

        let patchCountReached = await waitForPatchPinCount(expected: 1)
        XCTAssertTrue(patchCountReached)

        XCTAssertTrue(pinInfo.waitForDefaultMode())
        let patchBody = await tripResponseFactory.lastPinPatchBody()
        XCTAssertEqual(patchBody?.name, updatedName)
    }

    @MainActor
    func test_editPin_changeName_Fails() async throws {
        let tripInfo = TripInfoScreen(app: app)
        let pinInfo = PinInfoScreen(app: app)

        XCTAssertTrue(tripInfo.openTrip())
        XCTAssertTrue(tripInfo.tapPins())
        XCTAssertTrue(pinInfo.openPinFromPinsList(named: basePinName))

        XCTAssertTrue(pinInfo.tapEdit())

        let updatedName = "Updated Pin Name"
        pinInfo.setName(updatedName)
        XCTAssertTrue(pinInfo.tapDone())

        let patchCountReached = await waitForPatchPinCount(expected: 1)
        XCTAssertTrue(patchCountReached)

        XCTAssertTrue(
            pinInfo.waitForToast(
                [
                    PinzBaseStrings.PinInfo.Toast.saveFailed,
                    "Failed to save pin",
                    "Не удалось сохранить пин"
                ],
                timeout: 6
            )
        )
    }

    @MainActor
    func test_editPin_changeDescription_Succeeds() async throws {
        let tripInfo = TripInfoScreen(app: app)
        let pinInfo = PinInfoScreen(app: app)

        XCTAssertTrue(tripInfo.openTrip())
        XCTAssertTrue(tripInfo.tapPins())
        XCTAssertTrue(pinInfo.openPinFromPinsList(named: basePinName))

        XCTAssertTrue(pinInfo.tapEdit())

        let updatedDescription = "Updated pin description with useful details."
        pinInfo.setDescription(updatedDescription)
        XCTAssertTrue(pinInfo.tapDone())

        let patchCountReached = await waitForPatchPinCount(expected: 1)
        XCTAssertTrue(patchCountReached)

        XCTAssertTrue(pinInfo.waitForDefaultMode())
        XCTAssertTrue(pinInfo.waitForPinDescriptionValue(updatedDescription))

        XCTAssertTrue(
            pinInfo.waitForToast(
                [
                    PinzBaseStrings.PinInfo.Toast.pinSaved,
                    "Pin saved",
                    "Пин сохранён"
                ],
                timeout: 6
            )
        )

        let patchBody = await tripResponseFactory.lastPinPatchBody()
        XCTAssertEqual(patchBody?.description, updatedDescription)
    }

    @MainActor
    func test_deletePin_succeeds() async throws {
        let tripInfo = TripInfoScreen(app: app)
        let pinInfo = PinInfoScreen(app: app)

        XCTAssertTrue(tripInfo.openTrip())
        XCTAssertTrue(tripInfo.tapPins())
        XCTAssertTrue(pinInfo.openPinFromPinsList(named: basePinName))

        XCTAssertTrue(pinInfo.tapEdit())
        XCTAssertTrue(pinInfo.tapDeletePin())
        XCTAssertTrue(pinInfo.tapDeletePinConfirm())

        let deleteCountReached = await waitForDeletePinCount(expected: 1)
        XCTAssertTrue(deleteCountReached)

        let lastDeletedPinId = await tripResponseFactory.lastDeletedPinId()
        XCTAssertEqual(lastDeletedPinId, testingPinId)

        XCTAssertTrue(
            pinInfo.waitForToast(
                [
                    PinzBaseStrings.PinInfo.Toast.pinDeleted,
                    "Pin deleted",
                    "Пин удалён"
                ],
                timeout: 6
            )
        )
        XCTAssertTrue(pinInfo.waitForPinInfoToClose())
    }

    private func waitForPatchPinCount(expected: Int, timeout: TimeInterval = 2.0) async -> Bool {
        guard let tripResponseFactory else {
            return false
        }

        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if await tripResponseFactory.patchPinCount() == expected {
                return true
            }
            try? await Task.sleep(for: .milliseconds(100))
        }
        return false
    }

    private func waitForDeletePinCount(expected: Int, timeout: TimeInterval = 2.0) async -> Bool {
        guard let tripResponseFactory else {
            return false
        }

        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            if await tripResponseFactory.deletePinCount() == expected {
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
