import XCTest
import Foundation
@testable import PinzTrips
import PinzBase
import PinzDomain

final class TripInfoViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockNetwork: MockNetworkService!
    private var sut: TripInfoViewModel!
    private let trip = Trip.stub()

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockNetwork = MockNetworkService()
        sut = TripInfoViewModel(trip: trip, networkService: mockNetwork)
        sut.setRouter(mockRouter)
        SelectedTripStorage.shared.clearSelection()
    }

    override func tearDown() {
        mockNetwork = nil
        sut = nil
        SelectedTripStorage.shared.clearSelection()
        super.tearDown()
    }

    func test_initialState() {
        XCTAssertEqual(sut.state, .default)
        XCTAssertEqual(sut.trip.id, trip.id)
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

    func test_cancel_restoresTripState() {
        let originalSeason = sut.trip.season
        let originalCategory = sut.trip.category
        let originalName = sut.trip.name

        sut.dispatch(.changeState)
        sut.trip.season = .winter
        sut.trip.category = .business
        sut.trip.name = "Изменённое имя"
        sut.dispatch(.changeState)

        XCTAssertEqual(sut.state, .default)
        XCTAssertEqual(sut.trip.season, originalSeason)
        XCTAssertEqual(sut.trip.category, originalCategory)
        XCTAssertEqual(sut.trip.name, originalName)
    }

    func test_setImage_updatesImage() {
        let image = UIImage()
        sut.dispatch(.setImage(image))
        XCTAssertNotNil(sut.trip.image)
    }

    func test_setImage_nil_doesNotClearExistingImage() {
        sut.trip.image = UIImage()
        sut.dispatch(.setImage(nil))
        XCTAssertNotNil(sut.trip.image)
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_navigate_pinsList_callsRouter() {
        sut.dispatch(.navigate(.pinsList))
        XCTAssertEqual(mockRouter.navigatedPinsList?.id, trip.id)
    }

    func test_navigate_selectPins_callsRouter() {
        sut.dispatch(.navigate(.selectPins))
        XCTAssertEqual(mockRouter.navigatedSelectablePinsList?.id, trip.id)
    }

    func test_editTrip_updatesTripAndSwitchesToDefaultState() async throws {
        sut.dispatch(.changeState)
        mockNetwork.updateTripResult = .success(
            TripDTO(
                id: trip.id,
                name: "Updated trip",
                description: "Updated description",
                category: "business",
                season: "winter",
                coverUrl: "https://example.com/cover.jpg",
                ownerUserId: "user-001",
                privacyLevel: "private",
                status: "published",
                isPublished: false,
                isGenerated: false,
                likesCount: 0,
                dislikesCount: 0,
                startDateUnix: 1_700_000_100,
                endDateUnix: 1_700_010_200,
                createdAtUnix: 1_700_000_000,
                updatedAtUnix: 1_700_020_000
            )
        )

        sut.trip.name = "Updated trip"
        sut.trip.description = "Updated description"
        sut.trip.category = .business
        sut.trip.season = .winter
        sut.trip.privacyLevel = "private"
        sut.trip.coverUrl = "https://example.com/cover.jpg"
        sut.trip.startDate = Date(timeIntervalSince1970: 1_700_000_100)
        sut.trip.endDate = Date(timeIntervalSince1970: 1_700_010_200)

        await sut.asyncDispatch(.editTrip)

        let request = mockNetwork.updateTripCall
        XCTAssertEqual(request?.id, trip.id)
        XCTAssertEqual(request?.name, "Updated trip")
        XCTAssertEqual(request?.description, "Updated description")
        XCTAssertEqual(request?.category, "business")
        XCTAssertEqual(request?.season, "winter")
        XCTAssertEqual(request?.privacyLevel, "private")
        XCTAssertEqual(request?.coverUrl, "https://example.com/cover.jpg")
        XCTAssertEqual(request?.startDateUnix, 1_700_000_100)
        XCTAssertEqual(request?.endDateUnix, 1_700_010_200)
        XCTAssertEqual(sut.state, .default)
        XCTAssertEqual(sut.trip.name, "Updated trip")
    }

    func test_editTrip_errorKeepsEditingState() async {
        sut.dispatch(.changeState)
        mockNetwork.updateTripResult = .failure(URLError(.badServerResponse))

        var didReceiveError = false
        await sut.asyncDispatch(.editTrip) { _ in
            didReceiveError = true
        }
        XCTAssertTrue(didReceiveError)
        XCTAssertEqual(sut.state, .editing)
    }

    func test_editTrip_callsUpdateCallback() async throws {
        let callbackExpectation = expectation(description: "Trip update callback called")
        let callbackTripInfoViewModel = TripInfoViewModel(
            trip: trip,
            networkService: mockNetwork,
            onTripUpdated: { callbackExpectation.fulfill() }
        )
        callbackTripInfoViewModel.dispatch(.changeState)

        mockNetwork.updateTripResult = .success(
            TripDTO(
                id: trip.id,
                name: "Updated trip",
                description: nil,
                category: "vacation",
                season: "summer",
                coverUrl: nil,
                ownerUserId: "user-001",
                privacyLevel: nil,
                status: "published",
                isPublished: false,
                isGenerated: false,
                likesCount: 0,
                dislikesCount: 0,
                startDateUnix: nil,
                endDateUnix: nil,
                createdAtUnix: 1_700_000_000,
                updatedAtUnix: 1_700_000_000
            )
        )

        await callbackTripInfoViewModel.asyncDispatch(.editTrip)
        wait(for: [callbackExpectation], timeout: 1.0)
    }

    func test_editTrip_mapsCustomCategoryAndAutumnSeason() async throws {
        sut.dispatch(.changeState)
        mockNetwork.updateTripResult = .success(
            TripDTO(
                id: trip.id,
                name: "Custom category",
                description: nil,
                category: "custom-category",
                season: "autumn",
                coverUrl: nil,
                ownerUserId: "user-001",
                privacyLevel: nil,
                status: "published",
                isPublished: false,
                isGenerated: false,
                likesCount: 0,
                dislikesCount: 0,
                startDateUnix: nil,
                endDateUnix: nil,
                createdAtUnix: 1_700_000_000,
                updatedAtUnix: 1_700_000_000
            )
        )

        sut.trip.category = .custom("custom-category")
        sut.trip.season = .autumn

        await sut.asyncDispatch(.editTrip)

        let request = mockNetwork.updateTripCall
        XCTAssertEqual(request?.category, "custom-category")
        XCTAssertEqual(request?.season, "autumn")
    }

    func test_asyncDispatch_updateNotifications_sendsSettingsRequest() async {
        await sut.asyncDispatch(.updateNotifications(enabled: true))

        XCTAssertEqual(mockNetwork.updateTripSettingsCall?.id, trip.id)
        XCTAssertEqual(mockNetwork.updateTripSettingsCall?.notificationsEnabled, true)
    }

    func test_asyncDispatch_updateNotifications_callsErrorCallbackOnFailure() async {
        mockNetwork.updateTripSettingsResult = .failure(URLError(.badServerResponse))
        var didReceiveError = false

        await sut.asyncDispatch(.updateNotifications(enabled: false)) { _ in
            didReceiveError = true
        }

        XCTAssertTrue(didReceiveError)
    }

    func test_asyncDispatch_leaveTrip_callsLeaveTripAndClearsSelectionAndNavigatesBack() async {
        SelectedTripStorage.shared.selectedTripID = trip.id

        await sut.asyncDispatch(.leaveTrip)

        XCTAssertEqual(mockNetwork.leaveTripCall, trip.id)
        XCTAssertNil(SelectedTripStorage.shared.selectedTripID)
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_asyncDispatch_leaveTrip_callsErrorCallbackOnFailure() async {
        mockNetwork.leaveTripResult = .failure(URLError(.badServerResponse))
        SelectedTripStorage.shared.selectedTripID = trip.id
        var didReceiveError = false

        await sut.asyncDispatch(.leaveTrip) { _ in
            didReceiveError = true
        }

        XCTAssertTrue(didReceiveError)
        XCTAssertEqual(mockNetwork.leaveTripCall, trip.id)
        XCTAssertEqual(SelectedTripStorage.shared.selectedTripID, trip.id)
        XCTAssertEqual(mockRouter.popCallCount, 0)
    }
}
