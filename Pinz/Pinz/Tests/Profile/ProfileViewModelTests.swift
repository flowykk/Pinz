import XCTest
@testable import PinzProfile
import PinzBase
import PinzDomain
import UIKit

final class ProfileViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockNetwork: MockNetworkService!
    private var sut: ProfileViewModel!

    private let testUser = User(nickname: "tester", email: "test@example.com")

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockNetwork = MockNetworkService()
        sut = ProfileViewModel(user: testUser, networkService: mockNetwork)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        mockNetwork = nil
        sut = nil
        super.tearDown()
    }

    func test_initialState() {
        XCTAssertEqual(sut.state, .default)
        XCTAssertEqual(sut.user.nickname, "tester")
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

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    func test_navigate_statistics_callsRouter() {
        sut.dispatch(.navigate(.statistics))
        XCTAssertTrue(mockRouter.navigatedToStatistics)
    }

    func test_navigate_trips_callsRouter() {
        sut.dispatch(.navigate(.trips))
        XCTAssertTrue(mockRouter.navigatedToTrips)
    }

    func test_navigate_wishlist_callsRouter() {
        sut.dispatch(.navigate(.wishlist))
        XCTAssertTrue(mockRouter.navigatedToPlacesWishlist)
    }

    func test_navigate_saved_callsRouter() {
        sut.dispatch(.navigate(.saved))
        XCTAssertTrue(mockRouter.navigatedToSavedMaps)
    }

    func test_navigate_notifications_callsRouter() {
        sut.dispatch(.navigate(.notifications))
        XCTAssertTrue(mockRouter.navigatedToNotifications)
    }

    func test_navigate_appearance_callsRouter() {
        sut.dispatch(.navigate(.appearance))
        XCTAssertTrue(mockRouter.navigatedToAppearance)
    }

    func test_navigate_emailChange_callsRouterWithEmail() {
        sut.dispatch(.navigate(.emailChange))
        XCTAssertEqual(mockRouter.navigatedEmailChange?.email, testUser.email)
    }

    @MainActor
    func test_setImage_andSaveProfile_updatesAvatarAndNotifiesProfileUpdate() async throws {
        mockNetwork.requestAvatarUploadResult = .success(
            AvatarUploadResponseDTO(
                uploadUrl: "https://storage.example.com/avatar-upload",
                s3Key: "avatar-key"
            )
        )
        mockNetwork.confirmAvatarUploadResult = .success(
            ProfileResponseDTO(
                userId: "user-1",
                nickname: "tester",
                email: testUser.email,
                avatarUrl: "https://cdn.example.com/avatar-v2.jpg"
            )
        )
        mockNetwork.updateProfileResult = .success(
            ProfileResponseDTO(
                userId: "user-1",
                nickname: "updated-name",
                email: testUser.email,
                avatarUrl: "https://cdn.example.com/avatar-v2.jpg"
            )
        )

        sut.user.nickname = "updated-name"
        sut.dispatch(.setImage(makeTestImage()))
        sut.dispatch(.saveProfile)

        for _ in 0..<60 {
            if mockNetwork.requestAvatarUploadCall != nil,
               mockNetwork.confirmAvatarUploadCall != nil,
               mockRouter.currentProfileUpdateUser != nil {
                break
            }
            try await Task.sleep(nanoseconds: 20_000_000)
        }

        XCTAssertEqual(mockNetwork.requestAvatarUploadCall?.contentType, "image/jpeg")
        XCTAssertEqual(mockNetwork.confirmAvatarUploadCall, "avatar-key")
        XCTAssertEqual(mockNetwork.uploadToS3Call?.url, "https://storage.example.com/avatar-upload")
        XCTAssertEqual(sut.user.nickname, "updated-name")
        XCTAssertEqual(sut.user.avatarUrl, "https://cdn.example.com/avatar-v2.jpg")
        XCTAssertEqual(mockRouter.currentProfileUpdateUser?.avatarUrl, "https://cdn.example.com/avatar-v2.jpg")
        XCTAssertNil(sut.userImage)
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
