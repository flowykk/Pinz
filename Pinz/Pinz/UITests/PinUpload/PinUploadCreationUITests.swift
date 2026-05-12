import Foundation
import XCTest
import PinzBase
import PinzDomain

@MainActor
final class PinUploadCreationUITests: XCTestCase {
    private var app: XCUIApplication!
    private var backend: MockBackend!
    private var tripResponseFactory: TripInfoResponseFactory!
    private var profileResponseFactory: ProfileResponseFactory!
    private var pinUploadResponseFactory: PinUploadResponseFactory!

    private let testingTripId = "trip-ui-pin-upload-001"
    private let baseTripName = "UI Trip (Pin Upload)"
    private let baseTripDescription = "Trip created for PinUpload UI tests."
    private let existingPinId = "pin-ui-pin-upload-existing-001"
    private let existingPinName = "Existing Upload Pin"

    @MainActor
    override func setUp() {
        super.setUp()
        continueAfterFailure = false

        let now = Int(Date().timeIntervalSince1970)
        let existingPin = TripPinDTO(
            id: existingPinId,
            tripId: testingTripId,
            name: existingPinName,
            description: "Pin used for adding media through PinUpload.",
            category: "sight",
            latitude: 55.7558,
            longitude: 37.6176,
            startTimeUnix: nil,
            endTimeUnix: nil,
            tags: [],
            privacyLevel: "private",
            media: [
                TripPinMediaDTO(
                    mediaId: "pin-upload-existing-media-001",
                    url: "https://example.com/existing-pin-media.jpg",
                    mediaType: "image",
                    privacyLevel: "private",
                    capturedAtUnix: nil
                )
            ]
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
                initialPins: [existingPin]
            )
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
        pinUploadResponseFactory = PinUploadResponseFactory(tripId: testingTripId)

        do {
            backend = try MockBackend { routes in
                try routes.register(collection: ProfileController(responseFactory: profileResponseFactory))
                try routes.register(collection: TripInfoController(responseFactory: tripResponseFactory))
                try routes.register(collection: PinUploadController(responseFactory: pinUploadResponseFactory))
            }
        } catch {
            XCTFail("Failed to start pin upload mock backend: \(error)")
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
            testingTripId,
            PinzLaunchArg.testingPinUploadFakeMedia
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
        pinUploadResponseFactory = nil
        super.tearDown()
    }

    @MainActor
    func test_createPin_succeeds() async throws {
        let tripInfo = TripInfoScreen(app: app)
        let pinUpload = PinUploadScreen(app: app)

        XCTAssertTrue(tripInfo.openTrip())
        XCTAssertTrue(tripInfo.tapPins())
        XCTAssertTrue(pinUpload.tapAddPin())
        XCTAssertTrue(pinUpload.tapNext())

        let createdPinName = "Created UI Pin"
        pinUpload.setName(createdPinName)
        XCTAssertTrue(pinUpload.tapSave())

        let finalizeCountReached = await waitForFinalizeCount(expected: 1)
        XCTAssertTrue(finalizeCountReached)

        let counts = await pinUploadResponseFactory.counts()
        XCTAssertEqual(counts.start, 1)
        XCTAssertEqual(counts.upload, 1)
        XCTAssertEqual(counts.commit, 1)
        XCTAssertEqual(counts.process, 1)
        XCTAssertEqual(counts.finalize, 1)

        let startBody = await pinUploadResponseFactory.lastStartRequest()
        XCTAssertEqual(startBody?.filesToUpload.count, 1)
        XCTAssertEqual(startBody?.filesToUpload.first?.contentType, "image/jpeg")

        let finalizeBody = await pinUploadResponseFactory.lastFinalizeRequest()
        XCTAssertEqual(finalizeBody?.name, createdPinName)
        XCTAssertEqual(finalizeBody?.category, "sight")
        XCTAssertEqual(finalizeBody?.tags, ["ui"])
        XCTAssertEqual(finalizeBody?.mediaToDelete, [])
        XCTAssertTrue(pinUpload.waitForUploadFlowToClose())
    }

    @MainActor
    func test_addMediaToExistingPin_succeeds() async throws {
        let tripInfo = TripInfoScreen(app: app)
        let pinInfo = PinInfoScreen(app: app)
        let pinUpload = PinUploadScreen(app: app)

        XCTAssertTrue(tripInfo.openTrip())
        XCTAssertTrue(tripInfo.tapPins())
        XCTAssertTrue(pinInfo.openPinFromPinsList(named: existingPinName))
        XCTAssertTrue(pinInfo.openGallery())
        XCTAssertTrue(pinUpload.tapAddMediaToExistingPin())
        XCTAssertTrue(pinUpload.tapNext())
        XCTAssertTrue(pinUpload.tapSave())

        let finalizeCountReached = await waitForFinalizeCount(expected: 1)
        XCTAssertTrue(finalizeCountReached)

        let counts = await pinUploadResponseFactory.counts()
        XCTAssertEqual(counts.start, 1)
        XCTAssertEqual(counts.upload, 1)
        XCTAssertEqual(counts.commit, 1)
        XCTAssertEqual(counts.process, 1)
        XCTAssertEqual(counts.finalize, 1)

        let startBody = await pinUploadResponseFactory.lastStartRequest()
        XCTAssertEqual(startBody?.targetPinId, existingPinId)
        XCTAssertEqual(startBody?.filesToUpload.count, 1)
        XCTAssertEqual(startBody?.filesToUpload.first?.contentType, "image/jpeg")

        let finalizeBody = await pinUploadResponseFactory.lastFinalizeRequest()
        XCTAssertEqual(finalizeBody?.mediaToDelete, [])
        XCTAssertTrue(pinUpload.waitForUploadFlowToClose())
        XCTAssertTrue(pinInfo.waitForGalleryMode())
    }

    @MainActor
    private func waitForFinalizeCount(expected: Int, timeout: TimeInterval = 4.0) async -> Bool {
        let deadline = Date().addingTimeInterval(timeout)
        while Date() < deadline {
            let counts = await pinUploadResponseFactory.counts()
            if counts.finalize == expected {
                return true
            }
            try? await Task.sleep(for: .milliseconds(100))
        }
        let counts = await pinUploadResponseFactory.counts()
        return counts.finalize == expected
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
