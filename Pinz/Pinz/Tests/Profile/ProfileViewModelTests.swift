import XCTest
@testable import PinzProfile
import PinzBase
import PinzDomain
import UIKit

@MainActor
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

    // MARK: - setImage(nil)

    func test_dispatch_setImage_nil_doesNotSetUserImage() {
        sut.dispatch(.setImage(nil))
        XCTAssertNil(sut.userImage)
    }

    func test_dispatch_setImage_nil_doesNotStartAvatarUpload() async throws {
        sut.dispatch(.setImage(nil))
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertNil(mockNetwork.requestAvatarUploadCall)
    }

    // MARK: - getProfile

    func test_dispatch_getProfile_success_updatesUser() async throws {
        mockNetwork.getProfileResult = .success(ProfileResponseDTO(nickname: "updated-nick", email: "new@example.com"))
        sut.dispatch(.getProfile)
        try await waitForNotLoading()
        XCTAssertEqual(sut.user.nickname, "updated-nick")
        XCTAssertEqual(sut.user.email, "new@example.com")
    }

    func test_dispatch_getProfile_success_clearsUserImage() async throws {
        sut.userImage = makeTestImage()
        sut.dispatch(.getProfile)
        try await waitForNotLoading()
        XCTAssertNil(sut.userImage)
    }

    func test_dispatch_getProfile_success_setsIsLoadingFalse() async throws {
        sut.dispatch(.getProfile)
        try await waitForNotLoading()
        XCTAssertFalse(sut.isLoading)
    }

    func test_dispatch_getProfile_failure_setsIsLoadingFalse() async throws {
        mockNetwork.getProfileResult = .failure(URLError(.badServerResponse))
        sut.dispatch(.getProfile)
        try await waitForNotLoading()
        XCTAssertFalse(sut.isLoading)
    }

    func test_dispatch_getProfile_failure_keepsCurrentUser() async throws {
        mockNetwork.getProfileResult = .failure(URLError(.badServerResponse))
        sut.dispatch(.getProfile)
        try await waitForNotLoading()
        XCTAssertEqual(sut.user.nickname, testUser.nickname)
    }

    func test_dispatch_getProfile_whileLoading_isIgnored() async throws {
        mockNetwork.getProfileResult = .success(ProfileResponseDTO(nickname: "should-not-appear"))
        sut.isLoading = true
        sut.dispatch(.getProfile)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertTrue(sut.isLoading)
        XCTAssertNotEqual(sut.user.nickname, "should-not-appear")
    }

    // MARK: - saveProfile — missing paths

    func test_dispatch_saveProfile_failure_setsIsLoadingFalse() async throws {
        mockNetwork.updateProfileResult = .failure(URLError(.badServerResponse))
        sut.dispatch(.saveProfile)
        try await waitForNotLoading()
        XCTAssertFalse(sut.isLoading)
    }

    func test_dispatch_saveProfile_failure_transitionsToDefaultState() async throws {
        mockNetwork.updateProfileResult = .failure(URLError(.badServerResponse))
        sut.dispatch(.changeState)
        sut.dispatch(.saveProfile)
        try await waitForNotLoading()
        XCTAssertEqual(sut.state, .default)
    }

    func test_dispatch_saveProfile_trimsNicknameWhitespace() async throws {
        sut.user.nickname = "  john  "
        sut.dispatch(.saveProfile)
        try await waitForNotLoading()
        XCTAssertEqual(mockNetwork.updateProfileCall, "john")
    }

    // MARK: - deleteAccount

    func test_dispatch_deleteAccount_success_navigatesToMain() async throws {
        sut.dispatch(.deleteAccount)
        for _ in 0..<60 {
            if mockRouter.navigatedToMain { break }
            try await Task.sleep(nanoseconds: 20_000_000)
        }
        XCTAssertTrue(mockRouter.navigatedToMain)
    }

    func test_dispatch_deleteAccount_success_setsIsLoadingFalse() async throws {
        sut.dispatch(.deleteAccount)
        try await waitForNotLoading()
        XCTAssertFalse(sut.isLoading)
    }

    func test_dispatch_deleteAccount_failure_doesNotNavigate() async throws {
        mockNetwork.deleteAccountResult = .failure(URLError(.badServerResponse))
        sut.dispatch(.deleteAccount)
        try await waitForNotLoading()
        XCTAssertFalse(mockRouter.navigatedToMain)
    }

    func test_dispatch_deleteAccount_failure_setsIsLoadingFalse() async throws {
        mockNetwork.deleteAccountResult = .failure(URLError(.badServerResponse))
        sut.dispatch(.deleteAccount)
        try await waitForNotLoading()
        XCTAssertFalse(sut.isLoading)
    }

    func test_dispatch_deleteAccount_whileLoading_isIgnored() async throws {
        sut.isLoading = true
        sut.dispatch(.deleteAccount)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertTrue(sut.isLoading)
        XCTAssertFalse(mockRouter.navigatedToMain)
    }

    // MARK: - deleteAvatar

    func test_dispatch_deleteAvatar_success_updatesUser() async throws {
        mockNetwork.deleteAvatarResult = .success(ProfileResponseDTO(nickname: "avatar-deleted-nick", email: "test@example.com"))
        sut.dispatch(.deleteAvatar)
        try await waitForNotLoading()
        XCTAssertEqual(sut.user.nickname, "avatar-deleted-nick")
    }

    func test_dispatch_deleteAvatar_success_clearsUserImage() async throws {
        sut.userImage = makeTestImage()
        sut.dispatch(.deleteAvatar)
        try await waitForNotLoading()
        XCTAssertNil(sut.userImage)
    }

    func test_dispatch_deleteAvatar_success_notifiesProfileUpdate() async throws {
        sut.dispatch(.deleteAvatar)
        try await waitForNotLoading()
        XCTAssertNotNil(mockRouter.currentProfileUpdateUser)
    }

    func test_dispatch_deleteAvatar_failure_setsIsLoadingFalse() async throws {
        mockNetwork.deleteAvatarResult = .failure(URLError(.badServerResponse))
        sut.dispatch(.deleteAvatar)
        try await waitForNotLoading()
        XCTAssertFalse(sut.isLoading)
    }

    func test_dispatch_deleteAvatar_failure_keepsCurrentUser() async throws {
        mockNetwork.deleteAvatarResult = .failure(URLError(.badServerResponse))
        sut.dispatch(.deleteAvatar)
        try await waitForNotLoading()
        XCTAssertEqual(sut.user.nickname, testUser.nickname)
    }

    func test_dispatch_deleteAvatar_whileLoading_isIgnored() async throws {
        sut.isLoading = true
        sut.dispatch(.deleteAvatar)
        try await Task.sleep(nanoseconds: 50_000_000)
        XCTAssertTrue(sut.isLoading)
        XCTAssertNil(mockRouter.currentProfileUpdateUser)
    }

    // MARK: - Navigate: missing routes

    func test_navigate_storageSettings_callsRouter() {
        sut.dispatch(.navigate(.storageSettings))
        XCTAssertTrue(mockRouter.navigatedToStorageSettings)
    }

    func test_navigate_emailChange_actionCallback_updatesUserEmailAndPopsRouter() {
        sut.dispatch(.navigate(.emailChange))
        mockRouter.navigatedEmailChange?.action.action("changed@example.com")
        XCTAssertEqual(sut.user.email, "changed@example.com")
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    // MARK: - Helpers

    private func waitForNotLoading() async throws {
        for _ in 0..<60 {
            if !sut.isLoading { return }
            try await Task.sleep(nanoseconds: 20_000_000)
        }
        XCTFail("isLoading did not become false in time")
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
