import XCTest
import Foundation
@testable import PinzTrips
import PinzBase
import PinzDomain
import PinzNetworking
import UIKit

@MainActor
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

    @MainActor
    func test_photoBattle_startPhotoBattle_runs3RoundsAndSubmitsFinalWinner() async throws {
        let media = (1...TripInfoViewModel.requiredBattleMediaCount).map { index in
            StartBattleMediaDTO(
                photoBattleMediaId: "m-\(index)",
                mediaType: "photo",
                url: "https://example.com/\(index).jpg"
            )
        }
        mockNetwork.startBattleResult = .success(
            StartBattleResponseDTO(
                battleId: "battle-001",
                media: media
            )
        )

        await sut.startPhotoBattle()
        XCTAssertTrue(sut.isPhotoBattlePresented)
        XCTAssertEqual(mockNetwork.startBattleCall, trip.id)
        let battle = try XCTUnwrap(sut.photoBattleViewModel)
        XCTAssertEqual(battle.currentRound, 1)
        XCTAssertEqual(battle.currentPair?.0.photoBattleMediaId, "m-1")
        XCTAssertEqual(battle.currentPair?.1.photoBattleMediaId, "m-2")

        battle.selectPhotoBattleMedia(battle.leftMedia!)
        XCTAssertEqual(battle.step, 1)
        XCTAssertEqual(battle.currentPair?.0.photoBattleMediaId, "m-3")
        XCTAssertEqual(battle.currentPair?.1.photoBattleMediaId, "m-4")

        battle.selectPhotoBattleMedia(battle.leftMedia!)
        XCTAssertEqual(battle.step, 2)
        XCTAssertEqual(battle.currentPair?.0.photoBattleMediaId, "m-5")
        XCTAssertEqual(battle.currentPair?.1.photoBattleMediaId, "m-6")

        battle.selectPhotoBattleMedia(battle.leftMedia!)
        XCTAssertEqual(battle.step, 3)
        XCTAssertEqual(battle.currentPair?.0.photoBattleMediaId, "m-7")
        XCTAssertEqual(battle.currentPair?.1.photoBattleMediaId, "m-8")

        battle.selectPhotoBattleMedia(battle.leftMedia!)
        XCTAssertEqual(battle.step, 4)
        XCTAssertEqual(battle.currentRound, 2)
        XCTAssertEqual(battle.currentPair?.0.photoBattleMediaId, "m-1")
        XCTAssertEqual(battle.currentPair?.1.photoBattleMediaId, "m-3")

        battle.selectPhotoBattleMedia(battle.leftMedia!)
        XCTAssertEqual(battle.step, 5)
        XCTAssertEqual(battle.currentRound, 2)
        XCTAssertEqual(battle.currentPair?.0.photoBattleMediaId, "m-5")
        XCTAssertEqual(battle.currentPair?.1.photoBattleMediaId, "m-7")

        battle.selectPhotoBattleMedia(battle.leftMedia!)
        XCTAssertEqual(battle.step, 6)
        XCTAssertEqual(battle.currentRound, 3)
        XCTAssertEqual(battle.currentPair?.0.photoBattleMediaId, "m-1")
        XCTAssertEqual(battle.currentPair?.1.photoBattleMediaId, "m-5")

        battle.selectPhotoBattleMedia(battle.leftMedia!)

        for _ in 0..<80 {
            if mockNetwork.submitBattleResultCall != nil {
                break
            }
            try? await Task.sleep(nanoseconds: 10_000_000)
        }

        XCTAssertEqual(mockNetwork.submitBattleResultCall?.battleId, "battle-001")
        XCTAssertEqual(mockNetwork.submitBattleResultCall?.winnerMediaId, "m-1")
        XCTAssertNil(sut.battleError)
    }

    @MainActor
    func test_photoBattle_startPhotoBattle_preconditionFailureShowsError() async {
        mockNetwork.startBattleResult = .failure(HTTPError.preconditionFailed)

        await sut.startPhotoBattle()

        XCTAssertEqual(mockNetwork.startBattleCall, trip.id)
        XCTAssertEqual(sut.battleError, PinzBaseStrings.TripInfo.Message.photoBattleNeedMediaWithContext(TripInfoViewModel.requiredBattleMediaCount))
        XCTAssertFalse(sut.isPhotoBattlePresented)
    }

    @MainActor
    func test_photoBattle_submitFailureKeepsBattleOpenAndShowsError() async {
        let media = (1...TripInfoViewModel.requiredBattleMediaCount).map { index in
            StartBattleMediaDTO(
                photoBattleMediaId: "m-\(index)",
                mediaType: "photo",
                url: "https://example.com/\(index).jpg"
            )
        }
        mockNetwork.startBattleResult = .success(
            StartBattleResponseDTO(
                battleId: "battle-001",
                media: media
            )
        )
        mockNetwork.submitBattleResultResult = .failure(URLError(.badServerResponse))

        await sut.startPhotoBattle()
        XCTAssertTrue(sut.isPhotoBattlePresented)

        let battle = sut.photoBattleViewModel!
        battle.selectPhotoBattleMedia(battle.leftMedia!)
        battle.selectPhotoBattleMedia(battle.leftMedia!)
        battle.selectPhotoBattleMedia(battle.leftMedia!)
        battle.selectPhotoBattleMedia(battle.leftMedia!)
        battle.selectPhotoBattleMedia(battle.leftMedia!)
        battle.selectPhotoBattleMedia(battle.leftMedia!)
        battle.selectPhotoBattleMedia(battle.leftMedia!)

        for _ in 0..<80 {
            if battle.battleError != nil { break }
            try? await Task.sleep(nanoseconds: 10_000_000)
        }

        XCTAssertEqual(mockNetwork.submitBattleResultCall?.battleId, "battle-001")
        XCTAssertEqual(mockNetwork.submitBattleResultCall?.winnerMediaId, "m-1")
        XCTAssertNotNil(battle.battleError)
        XCTAssertTrue(sut.isPhotoBattlePresented)
    }

    @MainActor
    func test_photoBattle_startPhotoBattleBlockedWhenMediaCountLessThan8() async {
        let smallTrip = Trip(
            id: "trip-small",
            name: "Мало медиа",
            pins: [Pin(
                name: "Пикник",
                category: .entertainment,
                medias: (1...7).map { index in
                    MediaItem(
                        id: index,
                        isPrivate: false,
                        type: .image,
                        mediaURL: URL(string: "https://example.com/photo-\(index).jpg")
                    )
                },
                isPrivate: false,
                tags: []
            )],
            season: .summer,
            category: .vacation
        )
        let smallTripViewModel = TripInfoViewModel(trip: smallTrip, networkService: mockNetwork)

        await smallTripViewModel.startPhotoBattle()

        XCTAssertFalse(smallTripViewModel.canStartPhotoBattle)
        XCTAssertEqual(smallTripViewModel.battleError, PinzBaseStrings.TripInfo.Message.photoBattleNeedMedia(TripInfoViewModel.requiredBattleMediaCount))
        XCTAssertNil(mockNetwork.startBattleCall)
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

    @MainActor
    func test_setImage_uploadsTripCoverAndConfirmsUpload() async throws {
        mockNetwork.requestTripCoverUploadResult = .success(
            TripCoverUploadResponseDTO(
                uploadUrl: "https://storage.example.com/trip-cover-upload",
                s3Key: "trip-cover-key"
            )
        )
        mockNetwork.confirmTripCoverUploadResult = .success(
            TripDTO(
                id: trip.id,
                name: "Updated trip cover",
                description: nil,
                category: "vacation",
                season: "summer",
                coverUrl: "https://cdn.example.com/trip-cover.jpg",
                ownerUserId: "user-001",
                privacyLevel: "public",
                status: "published",
                isPublished: true,
                isGenerated: false,
                likesCount: 0,
                dislikesCount: 0,
                startDateUnix: nil,
                endDateUnix: nil,
                createdAtUnix: 1_700_000_000,
                updatedAtUnix: 1_700_000_000
            )
        )

        sut.dispatch(.setImage(makeTestImage()))

        for _ in 0..<60 {
            if mockNetwork.requestTripCoverUploadCall != nil,
               mockNetwork.confirmTripCoverUploadCall != nil {
                break
            }
            try await Task.sleep(nanoseconds: 20_000_000)
        }

        XCTAssertEqual(mockNetwork.requestTripCoverUploadCall?.id, trip.id)
        XCTAssertFalse(mockNetwork.requestTripCoverUploadCall?.filename.isEmpty ?? true)
        XCTAssertEqual(mockNetwork.confirmTripCoverUploadCall?.id, trip.id)
        XCTAssertEqual(mockNetwork.confirmTripCoverUploadCall?.s3Key, "trip-cover-key")
        XCTAssertEqual(mockNetwork.uploadToS3Call?.url, "https://storage.example.com/trip-cover-upload")
        XCTAssertEqual(sut.trip.coverUrl, "https://cdn.example.com/trip-cover.jpg")
        XCTAssertNil(sut.trip.image)
    }

    @MainActor
    func test_editTrip_usesUploadedTripCoverUrl() async throws {
        mockNetwork.requestTripCoverUploadResult = .success(
            TripCoverUploadResponseDTO(
                uploadUrl: "https://storage.example.com/trip-cover-upload",
                s3Key: "trip-cover-key"
            )
        )
        mockNetwork.confirmTripCoverUploadResult = .success(
            TripDTO(
                id: trip.id,
                name: "Updated trip cover",
                description: nil,
                category: "vacation",
                season: "summer",
                coverUrl: "https://cdn.example.com/trip-cover-final.jpg",
                ownerUserId: "user-001",
                privacyLevel: "public",
                status: "published",
                isPublished: true,
                isGenerated: false,
                likesCount: 0,
                dislikesCount: 0,
                startDateUnix: nil,
                endDateUnix: nil,
                createdAtUnix: 1_700_000_000,
                updatedAtUnix: 1_700_000_000
            )
        )
        mockNetwork.updateTripResult = .success(
            TripDTO(
                id: trip.id,
                name: "Updated trip",
                description: nil,
                category: "vacation",
                season: "summer",
                coverUrl: "https://cdn.example.com/update-trip-cover.jpg",
                ownerUserId: "user-001",
                privacyLevel: "public",
                status: "published",
                isPublished: true,
                isGenerated: false,
                likesCount: 0,
                dislikesCount: 0,
                startDateUnix: nil,
                endDateUnix: nil,
                createdAtUnix: 1_700_000_000,
                updatedAtUnix: 1_700_000_000
            )
        )

        sut.dispatch(.changeState)
        sut.dispatch(.setImage(makeTestImage()))

        await sut.asyncDispatch(.editTrip)

        XCTAssertEqual(mockNetwork.requestTripCoverUploadCall?.id, trip.id)
        XCTAssertEqual(mockNetwork.confirmTripCoverUploadCall?.id, trip.id)
        XCTAssertEqual(mockNetwork.updateTripCall?.coverUrl, "https://cdn.example.com/trip-cover-final.jpg")
        XCTAssertEqual(sut.trip.coverUrl, "https://cdn.example.com/update-trip-cover.jpg")
    }

    @MainActor
    func test_editTrip_stillSavesTripIfUploadFlowFails() async {
        mockNetwork.requestTripCoverUploadResult = .failure(URLError(.badServerResponse))
        mockNetwork.updateTripResult = .success(
            TripDTO(
                id: trip.id,
                name: "Updated trip",
                description: nil,
                category: "vacation",
                season: "summer",
                coverUrl: nil,
                ownerUserId: "user-001",
                privacyLevel: "public",
                status: "published",
                isPublished: true,
                isGenerated: false,
                likesCount: 0,
                dislikesCount: 0,
                startDateUnix: nil,
                endDateUnix: nil,
                createdAtUnix: 1_700_000_000,
                updatedAtUnix: 1_700_000_000
            )
        )

        sut.dispatch(.changeState)
        sut.dispatch(.setImage(makeTestImage()))

        await sut.asyncDispatch(.editTrip)

        XCTAssertEqual(mockNetwork.requestTripCoverUploadCall?.id, trip.id)
        XCTAssertNil(mockNetwork.confirmTripCoverUploadCall)
        XCTAssertNotNil(mockNetwork.updateTripCall)
        XCTAssertEqual(mockNetwork.updateTripCall?.coverUrl, nil)
        XCTAssertEqual(sut.state, .default)
    }

    @MainActor
    func test_editTrip_stillSavesTripIfUploadFlowConfirmFails() async {
        mockNetwork.requestTripCoverUploadResult = .success(
            TripCoverUploadResponseDTO(
                uploadUrl: "https://storage.example.com/trip-cover-upload",
                s3Key: "trip-cover-key"
            )
        )
        mockNetwork.confirmTripCoverUploadResult = .failure(URLError(.badServerResponse))
        mockNetwork.updateTripResult = .success(
            TripDTO(
                id: trip.id,
                name: "Updated trip",
                description: nil,
                category: "vacation",
                season: "summer",
                coverUrl: nil,
                ownerUserId: "user-001",
                privacyLevel: "public",
                status: "published",
                isPublished: true,
                isGenerated: false,
                likesCount: 0,
                dislikesCount: 0,
                startDateUnix: nil,
                endDateUnix: nil,
                createdAtUnix: 1_700_000_000,
                updatedAtUnix: 1_700_000_000
            )
        )

        sut.dispatch(.changeState)
        sut.dispatch(.setImage(makeTestImage()))

        await sut.asyncDispatch(.editTrip)

        XCTAssertEqual(mockNetwork.requestTripCoverUploadCall?.id, trip.id)
        XCTAssertEqual(mockNetwork.confirmTripCoverUploadCall?.id, trip.id)
        XCTAssertNotNil(mockNetwork.updateTripCall)
        XCTAssertNil(mockNetwork.updateTripCall?.coverUrl)
        XCTAssertEqual(sut.state, .default)
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

    private func makeTestImage() -> UIImage {
        let size = CGSize(width: 16, height: 16)
        let renderer = UIGraphicsImageRenderer(size: size)
        return renderer.image { context in
            UIColor.systemBlue.setFill()
            context.fill(CGRect(origin: .zero, size: size))
        }
    }
}
