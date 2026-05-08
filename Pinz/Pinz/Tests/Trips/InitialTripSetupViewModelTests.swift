import XCTest
@testable import PinzTrips
import PinzBase
import PinzDomain
import PinzNetworking

final class InitialTripSetupViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockNetwork: MockNetworkService!
    private var sut: InitialTripSetupViewModel!

    override func setUp() {
        super.setUp()
        let expectation = expectation(description: "setup main actor")
        Task {
            await MainActor.run {
                self.mockRouter = MockRouter()
                self.mockNetwork = MockNetworkService()
                self.sut = InitialTripSetupViewModel(networkService: self.mockNetwork)
                self.sut.setRouter(self.mockRouter)
                expectation.fulfill()
            }
        }
        wait(for: [expectation], timeout: 1.0)
    }

    override func tearDown() {
        let expectation = expectation(description: "teardown main actor")
        Task {
            await MainActor.run {
                self.mockRouter = nil
                self.mockNetwork = nil
                self.sut = nil
                expectation.fulfill()
            }
        }
        wait(for: [expectation], timeout: 1.0)
        super.tearDown()
    }

    // MARK: - Default state

    @MainActor
    func test_defaultState() {
        XCTAssertEqual(sut.state, .info)
        XCTAssertFalse(sut.isLoading)
        XCTAssertNil(sut.loadingStatus)
        XCTAssertTrue(sut.medias.isEmpty)
    }

    @MainActor
    func test_defaultName_isSet() {
        XCTAssertFalse(sut.name.isEmpty)
    }

    // MARK: - State.id and State.content

    func test_state_id_returnsItself() {
        XCTAssertEqual(InitialTripSetupViewModel.State.info.id, .info)
        XCTAssertEqual(InitialTripSetupViewModel.State.gallery.id, .gallery)
    }

    func test_state_content_info_isNotEmpty() {
        let content = InitialTripSetupViewModel.State.info.content
        if case .text(let text) = content {
            XCTAssertFalse(text.isEmpty)
        } else {
            XCTFail("Expected .text content for .info state")
        }
    }

    func test_state_content_gallery_isNotEmpty() {
        let content = InitialTripSetupViewModel.State.gallery.content
        if case .text(let text) = content {
            XCTAssertFalse(text.isEmpty)
        } else {
            XCTFail("Expected .text content for .gallery state")
        }
    }

    func test_state_content_infoDiffersFromGallery() {
        let info = InitialTripSetupViewModel.State.info.content
        let gallery = InitialTripSetupViewModel.State.gallery.content
        if case .text(let infoText) = info, case .text(let galleryText) = gallery {
            XCTAssertNotEqual(infoText, galleryText)
        }
    }

    // MARK: - LoadingStatus.localizedValue

    func test_loadingStatus_uploadingMedia_hasLocalizedValue() {
        let value = InitialTripSetupViewModel.LoadingStatus.uploadingMedia.localizedValue
        XCTAssertEqual(value, PinzBaseStrings.TripCreation.Loading.uploadingMedia)
    }

    func test_loadingStatus_formingPins_hasLocalizedValue() {
        let value = InitialTripSetupViewModel.LoadingStatus.formingPins.localizedValue
        XCTAssertEqual(value, PinzBaseStrings.TripCreation.Loading.formingPins)
    }

    // MARK: - deleteMedia

    @MainActor
    func test_dispatch_deleteMedia_removesMedia() {
        let id1 = UUID()
        let id2 = UUID()
        sut.medias = [
            LoadedMedia(id: id1, content: .loading),
            LoadedMedia(id: id2, content: .loading)
        ]
        sut.dispatch(.deleteMedia(id1))
        XCTAssertEqual(sut.medias.count, 1)
        XCTAssertEqual(sut.medias[0].id, id2)
    }

    @MainActor
    func test_dispatch_deleteMedia_unknownId_doesNothing() {
        let id = UUID()
        sut.medias = [LoadedMedia(id: id, content: .loading)]
        sut.dispatch(.deleteMedia(UUID()))
        XCTAssertEqual(sut.medias.count, 1)
    }

    // MARK: - Navigation

    @MainActor
    func test_dispatch_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    @MainActor
    func test_dispatch_navigate_preprocessedPins_callsRouter() {
        let pins = RawPins(pins: [RawPin(id: "p1", medias: [])])
        sut.dispatch(.navigate(.preprocessedPins(tripId: "trip-1", pins: pins)))
        XCTAssertEqual(mockRouter.navigatedTripCreationPreprocessedPins?.tripId, "trip-1")
        XCTAssertEqual(mockRouter.navigatedTripCreationPreprocessedPins?.pins.pins.count, 1)
    }

    // MARK: - asyncDispatch continue — success

    @MainActor
    func test_asyncDispatch_continue_success_navigatesToPreprocessedPins() async throws {
        let mediaId = UUID()
        sut.medias = [
            LoadedMedia(id: mediaId, content: .image(UIImage()), imageFileData: Data([1]), contentType: "image/jpeg")
        ]
        mockNetwork.createTripResult = .success(
            CreateTripDTO(tripId: "trip-new", status: "created", uploadUrls: [])
        )
        mockNetwork.processMediaGroupingResult = .success(
            ProcessMediaGroupingDTO(
                tripId: "trip-new",
                status: "processed",
                draftPins: [DraftPinDTO(draftPinId: "pin-1", media: [])]
            )
        )

        try await sut.asyncDispatch(.continue)

        XCTAssertEqual(mockRouter.navigatedTripCreationPreprocessedPins?.tripId, "trip-new")
        XCTAssertFalse(sut.isLoading)
    }

    @MainActor
    func test_asyncDispatch_continue_mapsNonEmptyDraftPins() async throws {
        let mediaId = UUID()
        sut.medias = [
            LoadedMedia(id: mediaId, content: .image(UIImage()), imageFileData: Data([1]), contentType: "image/jpeg")
        ]
        mockNetwork.createTripResult = .success(
            CreateTripDTO(tripId: "trip-x", status: "created", uploadUrls: [])
        )
        mockNetwork.processMediaGroupingResult = .success(
            ProcessMediaGroupingDTO(
                tripId: "trip-x",
                status: "processed",
                draftPins: [
                    DraftPinDTO(draftPinId: "pin-1", media: [
                        DraftPinMediaDTO(mediaId: "m1", type: "photo", url: "https://example.com/m1.jpg")
                    ]),
                    DraftPinDTO(draftPinId: "pin-2", media: [])
                ]
            )
        )

        try await sut.asyncDispatch(.continue)

        XCTAssertEqual(mockRouter.navigatedTripCreationPreprocessedPins?.pins.pins.count, 2)
    }

    @MainActor
    func test_asyncDispatch_continue_withNonLoadingMedia_sendsFilesToUpload() async throws {
        let mediaId = UUID()
        sut.medias = [
            LoadedMedia(id: mediaId, content: .image(UIImage()), imageFileData: Data([0, 1, 2]), contentType: "image/jpeg")
        ]
        mockNetwork.createTripResult = .success(
            CreateTripDTO(tripId: "trip-y", status: "created", uploadUrls: [])
        )
        mockNetwork.processMediaGroupingResult = .success(
            ProcessMediaGroupingDTO(
                tripId: "trip-y",
                status: "processed",
                draftPins: [DraftPinDTO(draftPinId: "pin-1", media: [])]
            )
        )

        try await sut.asyncDispatch(.continue)

        XCTAssertEqual(mockRouter.navigatedTripCreationPreprocessedPins?.tripId, "trip-y")
        XCTAssertFalse(sut.isLoading)
    }

    @MainActor
    func test_asyncDispatch_continue_withMatchingUploadUrl_uploadsToS3() async throws {
        let mediaId = UUID()
        sut.medias = [
            LoadedMedia(id: mediaId, content: makeTestImage(), imageFileData: Data([1, 2, 3, 4]), contentType: "image/jpeg")
        ]
        mockNetwork.createTripResult = .success(
            CreateTripDTO(
                tripId: "trip-z",
                status: "created",
                uploadUrls: [
                    UploadURLDTO(clientId: mediaId.uuidString, s3Key: "some-key", url: "https://s3.example.com/upload")
                ]
            )
        )
        mockNetwork.processMediaGroupingResult = .success(
            ProcessMediaGroupingDTO(tripId: "trip-z", status: "processed", draftPins: [])
        )

        try await sut.asyncDispatch(.continue)

        XCTAssertEqual(mockNetwork.uploadToS3Call?.url, "https://s3.example.com/upload")
        XCTAssertGreaterThan((mockNetwork.uploadToS3Call?.dataBytes ?? 0), 0)
        XCTAssertFalse(sut.isLoading)
    }

    @MainActor
    func test_asyncDispatch_continue_setsIsLoadingFalseAfterSuccess() async throws {
        mockNetwork.createTripResult = .success(CreateTripDTO(tripId: "t", status: "created", uploadUrls: []))
        mockNetwork.processMediaGroupingResult = .success(ProcessMediaGroupingDTO(tripId: "t", status: "ok", draftPins: []))

        try await sut.asyncDispatch(.continue)

        XCTAssertFalse(sut.isLoading)
        XCTAssertNil(sut.loadingStatus)
    }

    // MARK: - asyncDispatch continue — failures

    @MainActor
    func test_asyncDispatch_continue_createTripFailure_throws() async {
        let mediaId = UUID()
        sut.medias = [
            LoadedMedia(id: mediaId, content: .image(UIImage()), imageFileData: Data([1]), contentType: "image/jpeg")
        ]
        struct CreateError: Error {}
        mockNetwork.createTripResult = .failure(CreateError())

        do {
            try await sut.asyncDispatch(.continue)
            XCTFail("Expected error")
        } catch {
            XCTAssertTrue(error is CreateError)
        }
    }

    @MainActor
    func test_asyncDispatch_continue_processGroupingFailure_throws() async {
        let mediaId = UUID()
        sut.medias = [
            LoadedMedia(id: mediaId, content: .image(UIImage()), imageFileData: Data([1]), contentType: "image/jpeg")
        ]
        mockNetwork.createTripResult = .success(
            CreateTripDTO(tripId: "trip-x", status: "created", uploadUrls: [])
        )
        struct GroupError: Error {}
        mockNetwork.processMediaGroupingResult = .failure(GroupError())

        do {
            try await sut.asyncDispatch(.continue)
            XCTFail("Expected error")
        } catch {
            XCTAssertTrue(error is GroupError)
        }
    }

    @MainActor
    func test_asyncDispatch_continue_uploadToS3Failure_throws() async {
        let mediaId = UUID()
        sut.medias = [
            LoadedMedia(id: mediaId, content: .image(UIImage()), imageFileData: Data([1, 2, 3]), contentType: "image/jpeg")
        ]
        mockNetwork.createTripResult = .success(
            CreateTripDTO(
                tripId: "trip-fail",
                status: "created",
                uploadUrls: [UploadURLDTO(clientId: mediaId.uuidString, s3Key: "k", url: "https://s3.example.com/fail")]
            )
        )
        mockNetwork.uploadToS3Error = URLError(.badServerResponse)

        do {
            try await sut.asyncDispatch(.continue)
            XCTFail("Expected error from uploadToS3")
        } catch {
            XCTAssertTrue(error is URLError)
        }
    }

    @MainActor
    func test_asyncDispatch_continue_uploadMediaLimitExceededShowsToastAndThrows() async {
        var toasts: [String] = []
        sut.setToast { toasts.append($0) }

        let mediaId = UUID()
        sut.medias = [
            LoadedMedia(id: mediaId, content: .image(UIImage()), imageFileData: Data([1, 2, 3]), contentType: "image/jpeg")
        ]
        mockNetwork.createTripResult = .success(
            CreateTripDTO(
                tripId: "trip-over-limit",
                status: "created",
                uploadUrls: [
                    UploadURLDTO(
                        clientId: mediaId.uuidString,
                        s3Key: "s3-key",
                        url: "https://s3.example.com/over-limit"
                    )
                ]
            )
        )
        mockNetwork.uploadToS3Error = MediaUploadError.limitExceeded(
            kind: .image,
            originalBytes: 11_000_000,
            maxBytes: 10_000_000
        )

        do {
            try await sut.asyncDispatch(.continue)
            XCTFail("Expected error")
        } catch {
            XCTAssertTrue(error is MediaUploadError)
        }

        XCTAssertEqual(toasts, [MediaUploadPreprocessor.localizedLimitMessage(for: .image)])
    }

    @MainActor
    func test_asyncDispatch_continue_videoUsesUploadToS3FileAPI() async throws {
        let mediaId = UUID()
        let videoURL = makeTempMediaFile(data: Data([1, 2, 3]), filename: "temp-video")
        let firstFrame = makeTestImage()
        sut.medias = [
            LoadedMedia(id: mediaId, content: .video(url: videoURL, firstFrame: firstFrame))
        ]
        mockNetwork.createTripResult = .success(
            CreateTripDTO(
                tripId: "trip-video",
                status: "created",
                uploadUrls: [
                    UploadURLDTO(
                        clientId: mediaId.uuidString,
                        s3Key: "video-key",
                        url: "https://s3.example.com/video-upload"
                    )
                ]
            )
        )
        mockNetwork.processMediaGroupingResult = .success(
            ProcessMediaGroupingDTO(tripId: "trip-video", status: "processed", draftPins: [DraftPinDTO(draftPinId: "pin-1", media: [])])
        )

        try await sut.asyncDispatch(.continue)

        XCTAssertEqual(mockNetwork.uploadToS3FileURLCall?.url, "https://s3.example.com/video-upload")
        XCTAssertEqual(mockNetwork.uploadToS3FileURLCall?.contentType, "video/mp4")
    }

    private func makeTestImage() -> UIImage {
        let size = CGSize(width: 16, height: 16)
        let renderer = UIGraphicsImageRenderer(size: size)
        return renderer.image { context in
            UIColor.systemBlue.setFill()
            context.fill(CGRect(origin: .zero, size: size))
        }
    }

    private func makeTempMediaFile(data: Data, filename: String) -> URL {
        let fileURL = FileManager.default.temporaryDirectory
            .appendingPathComponent(filename)
            .appendingPathExtension("mp4")
        try? data.write(to: fileURL, options: .atomic)
        return fileURL
    }
}
