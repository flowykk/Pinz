public struct GenerateInviteLinkDTO: Codable {
    public let inviteLinkId: String
    public let inviteUrl: String?
    public let token: String
    public let expiresAtUnix: Int?

    public var effectiveInviteUrl: String {
        if let inviteUrl, !inviteUrl.isEmpty { return inviteUrl }
        return "https://pinz.website/join/\(token)"
    }

    public init(inviteLinkId: String, inviteUrl: String? = nil, token: String, expiresAtUnix: Int?) {
        self.inviteLinkId = inviteLinkId
        self.inviteUrl = inviteUrl
        self.token = token
        self.expiresAtUnix = expiresAtUnix
    }

    enum CodingKeys: String, CodingKey {
        case inviteLinkId = "invite_link_id"
        case inviteUrl = "invite_url"
        case token
        case expiresAtUnix = "expires_at_unix"
    }
}
