public struct GenerateInviteLinkDTO: Codable {
    public let inviteLinkId: String
    public let inviteUrl: String
    public let token: String
    public let expiresAtUnix: Int?

    public init(inviteLinkId: String, inviteUrl: String, token: String, expiresAtUnix: Int?) {
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
