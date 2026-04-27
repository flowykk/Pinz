import XCTest
import PinzBase
import PinzDomain
import PinzNetworking
@testable import PinzTrips

@MainActor
final class TripInviteViewModelTests: XCTestCase {

    private var mockNetwork: MockNetworkService!
    private var sut: TripInviteViewModel!

    override func setUp() {
        super.setUp()
        mockNetwork = MockNetworkService()
        mockNetwork.generateInviteLinkResult = .success(
            GenerateInviteLinkDTO(
                inviteLinkId: "link-001",
                inviteUrl: "https://pinz.website/join/token",
                token: "token",
                expiresAtUnix: nil
            )
        )
    }

    override func tearDown() {
        sut = nil
        mockNetwork = nil
        super.tearDown()
    }

    func test_load_success_setsInviteUrl() async {
        sut = TripInviteViewModel(tripId: "trip-1", networkService: mockNetwork)
        await sut.load()
        XCTAssertEqual(sut.inviteUrl, "https://pinz.website/join/token")
        XCTAssertNil(sut.errorMessage)
        XCTAssertFalse(sut.isLoading)
        XCTAssertEqual(mockNetwork.lastGenerateInviteLinkTripId, "trip-1")
        XCTAssertEqual(mockNetwork.generateInviteLinkCallCount, 1)
    }

    func test_load_failure_setsError() async {
        mockNetwork.generateInviteLinkResult = .failure(URLError(.notConnectedToInternet))
        sut = TripInviteViewModel(tripId: "trip-1", networkService: mockNetwork)
        await sut.load()
        XCTAssertNil(sut.inviteUrl)
        XCTAssertEqual(sut.errorMessage, PinzBaseStrings.TripMembers.Invite.error)
        XCTAssertFalse(sut.isLoading)
    }

    func test_retry_load_afterFailure_allowsSuccess() async {
        mockNetwork.generateInviteLinkResult = .failure(URLError(.notConnectedToInternet))
        sut = TripInviteViewModel(tripId: "trip-1", networkService: mockNetwork)
        await sut.load()
        XCTAssertNotNil(sut.errorMessage)
        mockNetwork.generateInviteLinkResult = .success(
            GenerateInviteLinkDTO(
                inviteLinkId: "l2",
                inviteUrl: "https://example.com/j",
                token: "t2",
                expiresAtUnix: nil
            )
        )
        await sut.load()
        XCTAssertEqual(sut.inviteUrl, "https://example.com/j")
        XCTAssertNil(sut.errorMessage)
    }

    func test_load_withoutInviteUrl_usesTokenPath() async {
        mockNetwork.generateInviteLinkResult = .success(
            GenerateInviteLinkDTO(
                inviteLinkId: "9975d18f-cc13-42e5-a617-46ecfff52992",
                inviteUrl: nil,
                token: "486817e2-8fa3-43ec-9678-27e0a31711cf",
                expiresAtUnix: 1777308716
            )
        )
        sut = TripInviteViewModel(tripId: "c5cd8a79-53c3-4a2b-9467-275c048a39df", networkService: mockNetwork)
        await sut.load()
        XCTAssertEqual(
            sut.inviteUrl,
            "https://pinz.website/join/486817e2-8fa3-43ec-9678-27e0a31711cf"
        )
    }
}
