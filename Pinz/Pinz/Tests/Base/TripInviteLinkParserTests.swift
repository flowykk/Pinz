import XCTest
import PinzBase

final class TripInviteLinkParserTests: XCTestCase {

    func test_https_pinzWebsite_extractsToken() {
        let url = URL(string: "https://pinz.website/join/abc-token")!
        XCTAssertEqual(TripInviteLinkParser.inviteToken(from: url), "abc-token")
    }

    func test_https_wwwPinzWebsite_extractsToken() {
        let url = URL(string: "https://www.pinz.website/join/tok-1")!
        XCTAssertEqual(TripInviteLinkParser.inviteToken(from: url), "tok-1")
    }

    func test_https_localhost_extractsToken() {
        let url = URL(string: "http://localhost:8080/join/dev-token")!
        XCTAssertEqual(TripInviteLinkParser.inviteToken(from: url), "dev-token")
    }

    func test_https_percentEncodedToken() {
        let url = URL(string: "https://pinz.website/join/hello%20world")!
        XCTAssertEqual(TripInviteLinkParser.inviteToken(from: url), "hello world")
    }

    func test_customScheme_pinz_joinHost_extractsToken() {
        let url = URL(string: "pinz://join/my-invite-token")!
        XCTAssertEqual(TripInviteLinkParser.inviteToken(from: url), "my-invite-token")
    }

    func test_randomUrl_returnsNil() {
        XCTAssertNil(TripInviteLinkParser.inviteToken(from: URL(string: "https://example.com/foo")!))
        XCTAssertNil(TripInviteLinkParser.inviteToken(from: URL(string: "pinz://open/settings")!))
        XCTAssertNil(TripInviteLinkParser.inviteToken(from: URL(string: "https://pinz.website/profile/x")!))
    }

    func test_https_missingJoinPath_returnsNil() {
        XCTAssertNil(TripInviteLinkParser.inviteToken(from: URL(string: "https://pinz.website/trips/1")!))
    }
}
