import XCTest
@testable import PinzProfile
import PinzBase
import PinzDomain

@MainActor
final class EmailChangeViewModelTests: XCTestCase {

    private var mockRouter: MockRouter!
    private var mockNetwork: MockNetworkService!
    private var sut: EmailChangeViewModel!
    private var successCallbackEmail: String?

    override func setUp() {
        super.setUp()
        mockRouter = MockRouter()
        mockNetwork = MockNetworkService()
        sut = EmailChangeViewModel(email: "old@example.com", networkService: mockNetwork) { [weak self] email in
            self?.successCallbackEmail = email
        }
        sut.setRouter(mockRouter)
    }

    override func tearDown() {
        mockNetwork = nil
        sut = nil
        successCallbackEmail = nil
        super.tearDown()
    }

    // MARK: - Initial state

    func test_initialState_isEmail() {
        XCTAssertEqual(sut.state, .email)
        XCTAssertEqual(sut.email, "old@example.com")
        XCTAssertFalse(sut.isLoading)
    }

    func test_initialCode_isAllEmpty() {
        XCTAssertEqual(sut.code, ["", "", "", ""])
    }

    // MARK: - isNextButtonDisabled

    func test_isNextButtonDisabled_emptyEmail_isTrue() {
        sut.email = ""
        XCTAssertTrue(sut.isNextButtonDisabled)
    }

    func test_isNextButtonDisabled_whitespaceOnlyEmail_isTrue() {
        sut.email = "   "
        XCTAssertTrue(sut.isNextButtonDisabled)
    }

    func test_isNextButtonDisabled_withEmail_isFalse() {
        sut.email = "test@example.com"
        XCTAssertFalse(sut.isNextButtonDisabled)
    }

    func test_isNextButtonDisabled_codeState_incompleteCode_isTrue() {
        sut.state = .code
        sut.code = ["1", "2", "3", ""]
        XCTAssertTrue(sut.isNextButtonDisabled)
    }

    func test_isNextButtonDisabled_codeState_fullCode_isFalse() {
        sut.state = .code
        sut.code = ["1", "2", "3", "4"]
        XCTAssertFalse(sut.isNextButtonDisabled)
    }

    // MARK: - nextButtonTitle

    func test_nextButtonTitle_emailState_returnsReceiveCode() {
        sut.state = .email
        XCTAssertEqual(sut.nextButtonTitle, PinzBaseStrings.EmailChange.Button.receiveCode)
    }

    func test_nextButtonTitle_codeState_returnsDone() {
        sut.state = .code
        XCTAssertEqual(sut.nextButtonTitle, PinzBaseStrings.Common.Button.done)
    }

    // MARK: - Navigation

    func test_navigate_back_callsPop() {
        sut.dispatch(.navigate(.back))
        XCTAssertEqual(mockRouter.popCallCount, 1)
    }

    // MARK: - continueTapped — guard

    func test_continueTapped_whileLoading_doesNotCallNetwork() async {
        sut.isLoading = true
        await sut.continueTapped()
        XCTAssertNil(mockNetwork.changeEmailCall)
    }

    // MARK: - requestCode — success

    func test_continueTapped_emailState_success_transitionsToCodeState() async {
        sut.email = "new@example.com"

        await sut.continueTapped()

        XCTAssertEqual(sut.state, .code)
    }

    func test_continueTapped_emailState_success_normalizesWhitespaceEmail() async {
        sut.email = "  new@example.com  "

        await sut.continueTapped()

        XCTAssertEqual(sut.email, "new@example.com")
        XCTAssertEqual(mockNetwork.changeEmailCall?.newEmail, "new@example.com")
    }

    func test_continueTapped_emailState_success_resetsCode() async {
        sut.code = ["1", "2", "3", "4"]

        await sut.continueTapped()

        XCTAssertEqual(sut.code, ["", "", "", ""])
    }

    func test_continueTapped_emailState_success_setsIsLoadingFalse() async {
        await sut.continueTapped()
        XCTAssertFalse(sut.isLoading)
    }

    // MARK: - requestCode — failure

    func test_continueTapped_emailState_failure_staysInEmailState() async {
        mockNetwork.changeEmailResult = .failure(URLError(.badServerResponse))

        await sut.continueTapped()

        XCTAssertEqual(sut.state, .email)
    }

    func test_continueTapped_emailState_failure_setsIsLoadingFalse() async {
        mockNetwork.changeEmailResult = .failure(URLError(.badServerResponse))

        await sut.continueTapped()

        XCTAssertFalse(sut.isLoading)
    }

    func test_continueTapped_emailState_emptyEmail_doesNotCallNetwork() async {
        sut.email = ""

        await sut.continueTapped()

        XCTAssertNil(mockNetwork.changeEmailCall)
        XCTAssertEqual(sut.state, .email)
    }

    // MARK: - requestUserId (via changeEmail call)

    func test_continueTapped_nilUserId_sendsNilToNetwork() async {
        sut = EmailChangeViewModel(email: "new@example.com", userId: nil, networkService: mockNetwork) { _ in }

        await sut.continueTapped()

        XCTAssertNil(mockNetwork.changeEmailCall?.userId)
    }

    func test_continueTapped_emptyUserId_sendsNilToNetwork() async {
        sut = EmailChangeViewModel(email: "new@example.com", userId: "", networkService: mockNetwork) { _ in }

        await sut.continueTapped()

        XCTAssertNil(mockNetwork.changeEmailCall?.userId)
    }

    func test_continueTapped_whitespaceUserId_sendsNilToNetwork() async {
        sut = EmailChangeViewModel(email: "new@example.com", userId: "   ", networkService: mockNetwork) { _ in }

        await sut.continueTapped()

        XCTAssertNil(mockNetwork.changeEmailCall?.userId)
    }

    func test_continueTapped_validUserId_sendsToNetwork() async {
        sut = EmailChangeViewModel(email: "new@example.com", userId: "user-123", networkService: mockNetwork) { _ in }

        await sut.continueTapped()

        XCTAssertEqual(mockNetwork.changeEmailCall?.userId, "user-123")
    }

    // MARK: - confirmCode — success

    func test_continueTapped_codeState_success_callsSuccessActionWithResponseEmail() async {
        mockNetwork.confirmEmailChangeResult = .success(ProfileResponseDTO(email: "confirmed@example.com"))
        sut.state = .code
        sut.code = ["1", "2", "3", "4"]

        await sut.continueTapped()

        XCTAssertEqual(successCallbackEmail, "confirmed@example.com")
    }

    func test_continueTapped_codeState_success_fallbacksToCurrentEmailWhenResponseEmailNil() async {
        mockNetwork.confirmEmailChangeResult = .success(ProfileResponseDTO(email: nil))
        sut.state = .code
        sut.email = "current@example.com"
        sut.code = ["1", "2", "3", "4"]

        await sut.continueTapped()

        XCTAssertEqual(successCallbackEmail, "current@example.com")
    }

    func test_continueTapped_codeState_success_sendsJoinedCodeToNetwork() async {
        sut.state = .code
        sut.code = ["5", "6", "7", "8"]

        await sut.continueTapped()

        XCTAssertEqual(mockNetwork.confirmEmailChangeCall, "5678")
    }

    func test_continueTapped_codeState_success_setsIsLoadingFalse() async {
        sut.state = .code
        sut.code = ["1", "2", "3", "4"]

        await sut.continueTapped()

        XCTAssertFalse(sut.isLoading)
    }

    // MARK: - confirmCode — failure

    func test_continueTapped_codeState_failure_doesNotCallSuccessAction() async {
        mockNetwork.confirmEmailChangeResult = .failure(URLError(.badServerResponse))
        sut.state = .code
        sut.code = ["1", "2", "3", "4"]

        await sut.continueTapped()

        XCTAssertNil(successCallbackEmail)
    }

    func test_continueTapped_codeState_failure_setsIsLoadingFalse() async {
        mockNetwork.confirmEmailChangeResult = .failure(URLError(.badServerResponse))
        sut.state = .code
        sut.code = ["1", "2", "3", "4"]

        await sut.continueTapped()

        XCTAssertFalse(sut.isLoading)
    }
}
