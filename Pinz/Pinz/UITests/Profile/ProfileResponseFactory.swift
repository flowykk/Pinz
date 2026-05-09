import Foundation
import Vapor

struct MockProfileSnapshot {
    var userId: String
    var username: String
    var nickname: String
    var email: String
    var avatarUrl: String?
}

struct MockProfileResponse: Content {
    let userId: String
    let username: String
    let nickname: String
    let email: String
    let avatarUrl: String?

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case username
        case nickname
        case email
        case avatarUrl = "avatar_url"
    }
}

struct MockChangeEmailResponse: Content {
    let success: Bool
    let message: String?
    let email: String?
    let expiresAtUnix: Int?

    enum CodingKeys: String, CodingKey {
        case success
        case message
        case email
        case expiresAtUnix = "expires_at_unix"
    }
}

struct MockUpdateProfileRequest: Content {
    let username: String
}

struct MockChangeEmailRequest: Content {
    let newEmail: String
    let userId: String?

    enum CodingKeys: String, CodingKey {
        case newEmail = "new_email"
        case userId = "user_id"
    }
}

struct MockConfirmEmailRequest: Content {
    let verificationCode: String

    enum CodingKeys: String, CodingKey {
        case verificationCode = "verification_code"
    }
}

actor MockProfileState {
    private let expectedEmailCode: String
    private var snapshot: MockProfileSnapshot
    private var profileRequests = 0
    private var updateRequests = 0
    private var requestEmailCount = 0
    private var confirmEmailCount = 0
    private var pendingEmail: String?

    init(snapshot: MockProfileSnapshot, expectedEmailCode: String = "1234") {
        self.snapshot = snapshot
        self.expectedEmailCode = expectedEmailCode
    }

    func current() async -> MockProfileSnapshot {
        profileRequests += 1
        return snapshot
    }

    func update(username: String) async -> MockProfileSnapshot {
        updateRequests += 1
        snapshot.username = username
        snapshot.nickname = username
        return snapshot
    }

    func requestEmailChange(newEmail: String, userId: String?) async -> MockProfileSnapshot {
        requestEmailCount += 1
        snapshot.userId = userId ?? snapshot.userId
        pendingEmail = newEmail
        return snapshot
    }

    func confirmEmailChange(code: String) async -> MockProfileSnapshot? {
        confirmEmailCount += 1
        guard code == expectedEmailCode else {
            return nil
        }
        guard let pendingEmail else {
            return snapshot
        }
        snapshot.email = pendingEmail
        snapshot.nickname = snapshot.nickname
        self.pendingEmail = nil
        return snapshot
    }

    func requestStats() async -> (
        profileRequests: Int,
        updateRequests: Int,
        requestEmailCount: Int,
        confirmEmailCount: Int
    ) {
        (profileRequests, updateRequests, requestEmailCount, confirmEmailCount)
    }
}

struct ProfileResponseFactory {
    private let state: MockProfileState

    init(
        initialProfile: MockProfileSnapshot = MockProfileSnapshot(
            userId: "user-123",
            username: "Flow",
            nickname: "Flow",
            email: "flow@example.com",
            avatarUrl: nil
        ),
        expectedEmailCode: String = "1234"
    ) {
        self.state = MockProfileState(snapshot: initialProfile, expectedEmailCode: expectedEmailCode)
    }

    func profileResponse() async -> Response {
        let snapshot = await state.current()
        return makeResponse(for: snapshot)
    }

    func updateProfileResponse(for request: MockUpdateProfileRequest) async -> Response {
        let snapshot = await state.update(username: request.username)
        return makeResponse(for: snapshot)
    }

    func requestEmailChangeResponse(for request: MockChangeEmailRequest) async -> Response {
        let snapshot = await state.requestEmailChange(newEmail: request.newEmail, userId: request.userId)
        return makeChangeEmailResponse(for: snapshot)
    }

    func confirmEmailChangeResponse(for request: MockConfirmEmailRequest) async -> Response? {
        guard let snapshot = await state.confirmEmailChange(code: request.verificationCode) else {
            return nil
        }
        return makeResponse(for: snapshot)
    }

    func patchRequestCount() async -> Int {
        await state.requestStats().updateRequests
    }

    func requestEmailCount() async -> Int {
        await state.requestStats().requestEmailCount
    }

    func confirmEmailCount() async -> Int {
        await state.requestStats().confirmEmailCount
    }

    private func makeResponse(for snapshot: MockProfileSnapshot) -> Response {
        let response = Response(status: .ok)
        try? response.content.encode(
            MockProfileResponse(
                userId: snapshot.userId,
                username: snapshot.username,
                nickname: snapshot.nickname,
                email: snapshot.email,
                avatarUrl: snapshot.avatarUrl
            )
        )
        return response
    }

    private func makeChangeEmailResponse(for snapshot: MockProfileSnapshot) -> Response {
        let response = Response(status: .ok)
        try? response.content.encode(
            MockChangeEmailResponse(
                success: true,
                message: "Verification code sent",
                email: snapshot.email,
                expiresAtUnix: 1700000000
            )
        )
        return response
    }
}
