import XCTest
@testable import PinzAuthentication
import PinzBase
import PinzDomain

@MainActor
final class AuthFlowViewModelTests: XCTestCase {

    private var mockNetwork: MockNetworkService!
    private var mockPasskey: MockPasskeyService!
    private var mockRouter: MockRouter!
    private var sut: AuthFlowViewModel!

    override func setUp() {
        super.setUp()
        mockNetwork = MockNetworkService()
        mockPasskey = MockPasskeyService()
        mockRouter = MockRouter()
        sut = AuthFlowViewModel(networkService: mockNetwork, passkeyService: mockPasskey)
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        sut = nil
        super.tearDown()
    }

    // MARK: - Initial state

    func test_initialState() {
        XCTAssertEqual(sut.state, .welcome)
        XCTAssertEqual(sut.longitude, 0)
        XCTAssertFalse(sut.isZoomedIn)
        XCTAssertEqual(sut.text, "")
    }

    // MARK: - Back navigation

    func test_back_fromEmail_resetsToWelcome() {
        sut.state = .email
        sut.dispatch(.back)
        XCTAssertEqual(sut.state, .welcome)
        XCTAssertFalse(sut.isZoomedIn)
    }

    func test_back_fromLoginPasskeyPrompt_goesToEmail() {
        sut.state = .login(.passkeyPrompt)
        sut.dispatch(.back)
        XCTAssertEqual(sut.state, .email)
    }

    func test_back_fromRegisterCode_goesToEmail() {
        sut.state = .register(.code)
        sut.dispatch(.back)
        XCTAssertEqual(sut.state, .email)
    }

    func test_back_fromRegisterNickname_goesToRegisterCode() {
        sut.state = .register(.nickname)
        sut.dispatch(.back)
        XCTAssertEqual(sut.state, .register(.code))
    }

    func test_back_fromRegisterPasskeyPrompt_goesToRegisterNickname() {
        sut.state = .register(.passkeyPrompt)
        sut.dispatch(.back)
        XCTAssertEqual(sut.state, .register(.nickname))
    }

    // MARK: - Welcome -> Email transition

    func test_proceedFromWelcome_setsIsZoomedIn() {
        sut.dispatch(.proceedFromWelcome)
        XCTAssertTrue(sut.isZoomedIn)
    }

    // MARK: - Rotation

    func test_updateRotation_incrementsLongitude() {
        let initial = sut.longitude
        sut.dispatch(.updateRotation)
        XCTAssertGreaterThan(sut.longitude, initial)
    }

    func test_updateRotation_wrapsAt180() {
        sut.longitude = 180
        sut.dispatch(.updateRotation)
        XCTAssertLessThan(sut.longitude, 0)
    }

    // MARK: - Async: Email submission

    func test_asyncContinue_fromEmail_unregisteredUser_goesToRegisterCode() async throws {
        mockNetwork.submitEmailResult = .success(
            SubmitEmailResponse(isRegistered: false, registrationId: "reg-001")
        )
        sut.state = .email
        sut.text = "new@example.com"

        try await sut.asyncDispatch(.continue)

        XCTAssertEqual(sut.state, .register(.code))
        XCTAssertEqual(sut.text, "")
    }

    func test_asyncContinue_fromEmail_registeredUser_simulator_navigatesToMain() async throws {
        // On simulator, performLogin skips passkey and goes straight to navigateToMain
        mockNetwork.submitEmailResult = .success(
            SubmitEmailResponse(isRegistered: true, registrationId: nil)
        )
        sut.state = .email
        sut.text = "existing@example.com"

        try await sut.asyncDispatch(.continue)

        #if targetEnvironment(simulator)
        XCTAssertTrue(mockRouter.navigatedToMain)
        #else
        XCTAssertEqual(sut.state, .login(.passkeyPrompt))
        #endif
    }

    func test_asyncContinue_fromEmail_networkError_throws() async {
        mockNetwork.submitEmailResult = .failure(URLError(.notConnectedToInternet))
        sut.state = .email
        sut.text = "test@example.com"

        do {
            try await sut.asyncDispatch(.continue)
            XCTFail("Expected error to be thrown")
        } catch {
            XCTAssertTrue(error is URLError)
        }
    }

    // MARK: - Async: Register code

    func test_asyncContinue_fromRegisterCode_success_goesToNickname() async throws {
        mockNetwork.verifyEmailResult = .success(SuccessResponse(success: true))
        sut.state = .register(.code)
        sut.text = "1234"

        try await sut.asyncDispatch(.continue)

        XCTAssertEqual(sut.state, .register(.nickname))
    }

    func test_asyncContinue_fromRegisterCode_failure_throws() async {
        mockNetwork.verifyEmailResult = .failure(URLError(.badServerResponse))
        sut.state = .register(.code)
        sut.text = "0000"

        do {
            try await sut.asyncDispatch(.continue)
            XCTFail("Expected error")
        } catch {
            XCTAssertTrue(error is URLError)
        }
    }

    // MARK: - Async: Register nickname -> passkeyPrompt

    func test_asyncContinue_fromRegisterNickname_simulator_navigatesToMain() async throws {
        sut.state = .register(.nickname)
        sut.text = "testuser"

        try await sut.asyncDispatch(.continue)

        #if targetEnvironment(simulator)
        XCTAssertTrue(mockRouter.navigatedToMain)
        #else
        XCTAssertEqual(sut.state, .register(.passkeyPrompt))
        #endif
    }
}
