import Foundation
import SwiftUI

public struct User: Hashable, Identifiable {
    public let id: UUID = UUID()
    public let profileId: String?
    public var nickname: String
    public var email: String
    public var avatarUrl: String?
    public var username: String?

    public init(
        nickname: String,
        email: String,
        avatarUrl: String? = nil,
        username: String? = nil,
        profileId: String? = nil
    ) {
        self.profileId = profileId
        self.nickname = nickname
        self.email = email
        self.avatarUrl = avatarUrl
        self.username = username
    }

    public init(profileDTO: ProfileResponseDTO) {
        self.profileId = profileDTO.id
        self.username = profileDTO.username
        self.nickname = profileDTO.nickname ?? profileDTO.username ?? ""
        self.email = profileDTO.email ?? ""
        self.avatarUrl = profileDTO.avatarUrl
    }
}

public extension ProfileResponseDTO {
    func toUser() -> User {
        User(profileDTO: self)
    }
}
