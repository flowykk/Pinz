import XCTest
@testable import PinzFeed
import PinzBase
import PinzDomain
import PinzNetworking

@MainActor
final class PostViewModelTests: XCTestCase {

    private var mockNetwork: MockNetworkService!
    private var sut: PostFeedItemViewModel!
    private var post: Post!

    override func setUp() {
        super.setUp()
        mockNetwork = MockNetworkService()
        post = Post.stub
        sut = PostFeedItemViewModel(post: post, networkService: mockNetwork)
    }

    override func tearDown() {
        mockNetwork = nil
        sut = nil
        post = nil
        super.tearDown()
    }

    // MARK: - Initial state

    func test_initialState() {
        XCTAssertFalse(sut.isLiked)
        XCTAssertFalse(sut.isDisliked)
        XCTAssertFalse(sut.isFavourite)
        XCTAssertTrue(sut.images.isEmpty)
    }

    func test_initialLikeCount_matchesPost() {
        XCTAssertEqual(sut.post.likes, post.likes)
    }

    // MARK: - Like

    func test_like_setsIsLiked_andIncrementCount() async throws {
        let initial = sut.post.likes
        sut.dispatch(.like)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertTrue(sut.isLiked)
        XCTAssertEqual(sut.post.likes, initial + 1)
    }

    func test_like_callsNetworkService() async throws {
        sut.dispatch(.like)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertEqual(mockNetwork.likeTripCall, post.id)
    }

    func test_like_whenAlreadyLiked_decrementsCount() async throws {
        sut.dispatch(.like)
        try await Task.sleep(nanoseconds: 50_000_000)
        let liked = sut.post.likes
        sut.dispatch(.like)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertFalse(sut.isLiked)
        XCTAssertEqual(sut.post.likes, liked - 1)
    }

    func test_like_whenDisliked_removesDislike() async throws {
        sut.dispatch(.dislike)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertTrue(sut.isDisliked)

        sut.dispatch(.like)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertTrue(sut.isLiked)
        XCTAssertFalse(sut.isDisliked)
    }

    // MARK: - Dislike

    func test_dislike_setsIsDisliked_andIncrementCount() async throws {
        let initial = sut.post.dislikes
        sut.dispatch(.dislike)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertTrue(sut.isDisliked)
        XCTAssertEqual(sut.post.dislikes, initial + 1)
    }

    func test_dislike_callsNetworkService() async throws {
        sut.dispatch(.dislike)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertEqual(mockNetwork.dislikeTripCall, post.id)
    }

    func test_dislike_whenAlreadyDisliked_decrementsCount() async throws {
        sut.dispatch(.dislike)
        try await Task.sleep(nanoseconds: 50_000_000)
        let disliked = sut.post.dislikes
        sut.dispatch(.dislike)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertFalse(sut.isDisliked)
        XCTAssertEqual(sut.post.dislikes, disliked - 1)
    }

    func test_dislike_whenLiked_removesLike() async throws {
        sut.dispatch(.like)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertTrue(sut.isLiked)

        sut.dispatch(.dislike)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertTrue(sut.isDisliked)
        XCTAssertFalse(sut.isLiked)
    }

    // MARK: - Favourite

    func test_toggleFavourite_add_setsFavourite() async throws {
        let initial = sut.post.favorites
        sut.dispatch(.toggleFavourite)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertTrue(sut.isFavourite)
        XCTAssertEqual(sut.post.favorites, initial + 1)
    }

    func test_toggleFavourite_add_callsAddToFavourites() async throws {
        sut.dispatch(.toggleFavourite)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertEqual(mockNetwork.addTripToFavouritesCall, post.id)
    }

    func test_toggleFavourite_remove_clearsFavourite() async throws {
        sut.dispatch(.toggleFavourite)
        try await Task.sleep(nanoseconds: 50_000_000)
        let faved = sut.post.favorites
        sut.dispatch(.toggleFavourite)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertFalse(sut.isFavourite)
        XCTAssertEqual(sut.post.favorites, faved - 1)
    }

    func test_toggleFavourite_remove_callsRemoveFromFavourites() async throws {
        sut.dispatch(.toggleFavourite)
        try await Task.sleep(nanoseconds: 50_000_000)
        sut.dispatch(.toggleFavourite)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertEqual(mockNetwork.removeTripFromFavouritesCall, post.id)
    }
}
