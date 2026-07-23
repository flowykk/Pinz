import Foundation
import XCTest
import PinzBase

@MainActor
final class ProfileEditingUITests: XCTestCase {

    private var app: XCUIApplication!
    private var backend: MockBackend!
    private var responseFactory: ProfileResponseFactory!

    private let initialNickname = "Flow"
    private let initialEmail = "flow@example.com"

    @MainActor
    override func setUp() {
        super.setUp()
        continueAfterFailure = false

        responseFactory = ProfileResponseFactory(
            initialProfile: MockProfileSnapshot(
                userId: "user-123",
                username: initialNickname,
                nickname: initialNickname,
                email: initialEmail,
                avatarUrl: nil
            )
        )

        do {
            backend = try MockBackend { routes in
                try routes.register(collection: ProfileController(responseFactory: responseFactory))
            }
        } catch {
            XCTFail("Failed to start profile mock backend: \(error)")
            return
        }

        backend.launch()
        XCTAssertTrue(waitForBackendHealth(timeout: 3.0))

        app = XCUIApplication()
        app.launchArguments = [
            PinzLaunchArg.useLocalhost,
            PinzLaunchArg.fakeTokens,
            PinzLaunchArg.testingProfile
        ]
        app.launch()
    }

    @MainActor
    override func tearDown() {
        app?.terminate()
        backend?.shutdown()
        app = nil
        backend = nil
        responseFactory = nil
        super.tearDown()
    }

    @MainActor
    func testProfile_EditNickname_Succeeds() async throws {
        let screen = ProfileScreen(app: app)

        XCTAssertTrue(screen.openProfile())
        XCTAssertTrue(screen.tapEdit())

        let updatedNickname = "pinz_user_01"
        screen.setNickname(updatedNickname)
        XCTAssertTrue(screen.tapSave())
        XCTAssertTrue(screen.waitForHeaderNickname("\(updatedNickname) • \(initialEmail)"))

        let patchCount = await responseFactory.patchRequestCount()
        XCTAssertEqual(patchCount, 1)
    }

    @MainActor
    func testProfile_EditNickname_ValidationTooShort() async throws {
        let screen = ProfileScreen(app: app)

        XCTAssertTrue(screen.openProfile())
        XCTAssertTrue(screen.tapEdit())

        screen.setNickname("ab")
        XCTAssertTrue(screen.tapSave())

        let patchCount = await responseFactory.patchRequestCount()
        XCTAssertEqual(patchCount, 0)
        XCTAssertTrue(screen.isInEditMode())
        XCTAssertTrue(screen.waitForValidationToast(
            [
                PinzBaseStrings.Profile.Toast.nicknameLengthInvalid,
                "Имя пользователя должно быть от 4 до 20 символов"
            ],
            timeout: 3.0
        ))
    }

    @MainActor
    func testProfile_EditEmail_Succeeds() async throws {
        let screen = ProfileScreen(app: app)
        let newEmail = "new@example.com"
        let verificationCode = "1234"

        XCTAssertTrue(screen.openProfile())
        XCTAssertTrue(screen.tapEdit())
        XCTAssertTrue(screen.tapChangeEmail())

        screen.setEmail(newEmail)
        XCTAssertTrue(screen.tapReceiveCode())
        let requestSent = await waitForEmailRequestCount(expected: 1)
        XCTAssertTrue(requestSent)
        XCTAssertTrue(screen.waitForVerificationCode(timeout: 5.0))

        screen.enterEmailCode(verificationCode)
        XCTAssertTrue(screen.tapCodeConfirm())

        XCTAssertTrue(screen.waitForHeaderEmail(newEmail))

        let confirmSent = await waitForEmailConfirmCount(expected: 1)
        XCTAssertTrue(confirmSent)
    }

    @MainActor
    func testProfile_EditEmail_InvalidEmail_ShowsValidationError() async throws {
        let screen = ProfileScreen(app: app)

        XCTAssertTrue(screen.openProfile())
        XCTAssertTrue(screen.tapEdit())
        XCTAssertTrue(screen.tapChangeEmail())

        screen.setEmail("invalid")
        XCTAssertTrue(screen.tapReceiveCode())

        XCTAssertTrue(screen.emailFieldExists(timeout: 2.0))
        XCTAssertTrue(screen.waitForValidationToast(
            [
                PinzBaseStrings.EmailChange.Toast.invalidEmail,
                "Введите корректный email"
            ],
            timeout: 3.0
        ))

        let requestCount = await responseFactory.requestEmailCount()
        let confirmCount = await responseFactory.confirmEmailCount()
        XCTAssertEqual(requestCount, 0)
        XCTAssertEqual(confirmCount, 0)
    }

    @MainActor
    func testProfile_EditEmail_InvalidCode_ShowsError() async throws {
        let screen = ProfileScreen(app: app)
        let newEmail = "new@example.com"

        XCTAssertTrue(screen.openProfile())
        XCTAssertTrue(screen.tapEdit())
        XCTAssertTrue(screen.tapChangeEmail())

        screen.setEmail(newEmail)
        XCTAssertTrue(screen.tapReceiveCode())
        XCTAssertTrue(screen.waitForVerificationCode(timeout: 5.0))

        let requestSent = await waitForEmailRequestCount(expected: 1)
        XCTAssertTrue(requestSent)

        screen.enterEmailCode("0000")
        XCTAssertTrue(screen.tapCodeConfirm())
        XCTAssertTrue(screen.waitForValidationToast(
            [
                PinzBaseStrings.EmailChange.Toast.confirmFailed,
                "Неверный код или срок его действия истёк"
            ],
            timeout: 3.0
        ))
        XCTAssertTrue(screen.waitForVerificationCode(timeout: 2.0))

        let confirmSent = await waitForEmailConfirmCount(expected: 1)
        XCTAssertTrue(confirmSent)
        XCTAssertFalse(screen.waitForHeaderEmail(newEmail))
    }

    @MainActor
    func testProfile_EditEmail_CancelFromCodeState_DoesNotConfirm() async throws {
        let screen = ProfileScreen(app: app)
        let newEmail = "new@example.com"

        XCTAssertTrue(screen.openProfile())
        XCTAssertTrue(screen.tapEdit())
        XCTAssertTrue(screen.tapChangeEmail())

        screen.setEmail(newEmail)
        XCTAssertTrue(screen.tapReceiveCode())
        XCTAssertTrue(screen.waitForVerificationCode(timeout: 5.0))
        XCTAssertTrue(screen.tapEmailChangeBack())

        let confirmCount = await responseFactory.confirmEmailCount()
        let requestSent = await waitForEmailRequestCount(expected: 1)
        XCTAssertTrue(requestSent)
        XCTAssertTrue(screen.isInEditMode())
        XCTAssertEqual(confirmCount, 0)
        XCTAssertFalse(screen.waitForHeaderEmail(newEmail))
    }

    @MainActor
    func testProfile_EditEmail_InvalidCodeLength_DoesNotCallConfirm() async throws {
        let screen = ProfileScreen(app: app)
        let newEmail = "new@example.com"

        XCTAssertTrue(screen.openProfile())
        XCTAssertTrue(screen.tapEdit())
        XCTAssertTrue(screen.tapChangeEmail())

        screen.setEmail(newEmail)
        XCTAssertTrue(screen.tapReceiveCode())
        XCTAssertTrue(screen.waitForVerificationCode(timeout: 5.0))
        let requestSent = await waitForEmailRequestCount(expected: 1)
        XCTAssertTrue(requestSent)

        screen.enterEmailCode("12")
        XCTAssertTrue(screen.tapCodeConfirm())

        let confirmCount = await responseFactory.confirmEmailCount()
        let requestSentAfterCode = await waitForEmailRequestCount(expected: 1)
        XCTAssertTrue(requestSentAfterCode)
        XCTAssertFalse(screen.waitForHeaderEmail(newEmail))
        XCTAssertEqual(confirmCount, 0)
    }

    private func waitForEmailRequestCount(expected: Int, timeout: TimeInterval = 2.0) async -> Bool {
        await waitUntil(timeout: timeout) {
            await self.responseFactory.requestEmailCount() == expected
        }
    }

    private func waitForEmailConfirmCount(expected: Int, timeout: TimeInterval = 2.0) async -> Bool {
        await waitUntil(timeout: timeout) {
            await self.responseFactory.confirmEmailCount() == expected
        }
    }
}
