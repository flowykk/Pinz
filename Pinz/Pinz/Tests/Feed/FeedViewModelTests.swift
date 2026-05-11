import XCTest
@testable import PinzFeed
import PinzDomain
import PinzNetworking

@MainActor
final class FeedViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockNetwork: MockNetworkService!
    private var sut: FeedViewModel!

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockNetwork = MockNetworkService()
        sut = FeedViewModel(networkService: mockNetwork)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        mockNetwork = nil
        sut = nil
        super.tearDown()
    }

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_loadRecommendation_insertsRecommendedPostAtTopAndHidesButton() async throws {
        sut.filters.city = "Париж"
        let recommendation = makeRecommendationResponse(city: "Париж")
        mockNetwork.getRecommendationsResult = .success(recommendation)

        await sut.fetchFeed()
        let feedCount = sut.posts.count

        sut.requestRecommendationsButtonTapped()
        try await Task.sleep(nanoseconds: 100_000_000)

        XCTAssertEqual(mockNetwork.getRecommendationsCall?.city, "Париж")
        XCTAssertNil(mockNetwork.getRecommendationsCall?.country)
        XCTAssertEqual(sut.posts.count, feedCount + 1)
        XCTAssertTrue(sut.posts.first?.isRecommended == true)
        XCTAssertEqual(sut.posts.first?.id, recommendation.map.trip.id)
        XCTAssertEqual(sut.posts.first?.recommendedBadge, "Рекомендация для тебя")
        XCTAssertFalse(sut.shouldShowRecommendationButton)
    }

    func test_loadRecommendation_withCountryOnly_callsNetworkWithCountry() async throws {
        sut.filters.city = ""
        sut.filters.country = "Италия"
        let recommendation = makeRecommendationResponse(city: "Италия")
        mockNetwork.getRecommendationsResult = .success(recommendation)

        await sut.fetchFeed()

        sut.requestRecommendationsButtonTapped()
        try await Task.sleep(nanoseconds: 100_000_000)

        XCTAssertEqual(mockNetwork.getRecommendationsCall?.city, nil)
        XCTAssertEqual(mockNetwork.getRecommendationsCall?.country, "Италия")
    }

    func test_loadRecommendation_doesNotCallWhenNoLocationAndShowsToast() async throws {
        var toasts: [String] = []
        sut.setToast { message in
            toasts.append(message)
        }

        sut.requestRecommendationsButtonTapped()
        try await Task.sleep(nanoseconds: 100_000_000)

        XCTAssertNil(mockNetwork.getRecommendationsCall)
        XCTAssertEqual(toasts, ["Выберите город или страну для рекомендаций."])
        XCTAssertTrue(sut.shouldShowRecommendationButton)
    }

    func test_loadRecommendation_doesNotCallWhenBothCityAndCountrySetAndShowsToast() async throws {
        var toasts: [String] = []
        sut.setToast { message in
            toasts.append(message)
        }
        sut.filters.city = "Париж"
        sut.filters.country = "Франция"

        sut.requestRecommendationsButtonTapped()
        try await Task.sleep(nanoseconds: 100_000_000)

        XCTAssertNil(mockNetwork.getRecommendationsCall)
        XCTAssertEqual(toasts, ["Выберите город или страну для рекомендаций."])
        XCTAssertTrue(sut.shouldShowRecommendationButton)
    }

    func test_loadMore_doesNotMoveRecommendationFromTop() async throws {
        let feedPage = mockNetwork.getFeedResult
        let feedPage2 = Result<[FeedItemDTO], Error>.success([
            FeedItemDTO(
                trip: TripDTO(
                    id: "trip-feed-extra",
                    name: "Доп. пост",
                    description: "Ещё один пост",
                    category: "vacation",
                    season: "spring",
                    coverUrl: nil,
                    ownerUserId: "user-extra",
                    privacyLevel: "public",
                    status: "published",
                    isPublished: true,
                    isGenerated: false,
                    likesCount: 10,
                    dislikesCount: 1,
                    participantsCount: 2,
                    mediaCount: 2,
                    startDateUnix: 1_700_000_000,
                    endDateUnix: 1_700_020_000,
                    createdAtUnix: 1_699_990_000,
                    updatedAtUnix: 1_699_990_000
                ),
                pins: [],
                media: []
            )
        ])

        mockNetwork.getRecommendationsResult = .success(makeRecommendationResponse(city: "Париж"))
        mockNetwork.getFeedResult = feedPage
        await sut.fetchFeed()
        sut.filters.city = "Париж"
        sut.requestRecommendationsButtonTapped()
        try await Task.sleep(nanoseconds: 100_000_000)

        let countAfterRecommendation = sut.posts.count
        mockNetwork.getFeedResult = feedPage2
        await sut.loadMore()

        XCTAssertEqual(sut.posts.first?.isRecommended, true)
        XCTAssertEqual(sut.posts.count, countAfterRecommendation + 1)
        XCTAssertEqual(sut.posts[1].id, "trip-feed-001")
        XCTAssertEqual(sut.posts.last?.id, "trip-feed-extra")
    }

    private func makeRecommendationResponse(city: String) -> GetRecommendationsResponseDTO {
        GetRecommendationsResponseDTO(
            map: RecommendedMapDTO(
                media: [
                    FeedMediaDTO(
                        mediaId: "rec-media-001",
                        url: "https://example.com/rec.jpg",
                        mediaType: "photo"
                    )
                ],
                pins: [
                    RecommendedPinDTO(
                        id: "rec-pin-001",
                        tripId: "trip-rec",
                        name: "Рекомендованный пин",
                        description: "Коротко",
                        category: "vacation",
                        latitude: 1.0,
                        longitude: 2.0,
                        locationName: city,
                        mediaCount: 1,
                        media: [
                            FeedMediaDTO(
                                mediaId: "rec-pin-media-001",
                                url: "https://example.com/rec-pin.jpg",
                                mediaType: "photo"
                            )
                        ]
                    )
                ],
                regionName: city,
                regionType: "city",
                snapshotToken: "stub-snapshot-token-001",
                trip: TripDTO(
                    id: "trip-rec",
                    name: "Рекомендованный маршрут",
                    description: "Тур с акцентом на лучшее в локации",
                    category: "vacation",
                    season: "spring",
                    coverUrl: nil,
                    ownerUserId: "user-rec",
                    privacyLevel: "public",
                    status: "published",
                    isPublished: true,
                    isGenerated: false,
                    likesCount: 12,
                    dislikesCount: 2,
                    participantsCount: 2,
                    mediaCount: 4,
                    startDateUnix: 1_700_000_000,
                    endDateUnix: 1_700_020_000,
                    createdAtUnix: 1_699_990_000,
                    updatedAtUnix: 1_699_990_000
                )
            )
        )
    }
}
