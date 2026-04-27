public struct GenerateInviteLinkDTO: Codable {
    public let inviteLinkId: String
    /// Omitted by some API deployments; use ``effectiveInviteUrl`` for display / sharing.
    public let inviteUrl: String?
    public let token: String
    public let expiresAtUnix: Int?

    /// When the server omits `invite_url`, builds `https://pinz.website/join/{token}` (same path shape as the networking stub for this endpoint).
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
