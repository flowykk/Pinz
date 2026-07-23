import Foundation
import SwiftUI

public struct TripMember: Hashable, Identifiable {
    public let id: UUID = UUID()
    public let isPrivate: Bool
    public let username: String
    public let avatar: UIImage

    public init(isPrivate: Bool, username: String, avatar: UIImage) {
        self.isPrivate = isPrivate
        self.username = username
        self.avatar = avatar
    }
}

extension TripMember {
    public static func stubs() -> [TripMember] {
        [
            TripMember(
                isPrivate: true,
                username: "danuwka",
                avatar: PinzDomainAsset.defaultPlaceholder.image
            ),
            TripMember(
                isPrivate: false,
                username: "kostik",
                avatar: PinzDomainAsset.groupPlaceholder.image
            ),
            TripMember(
                isPrivate: false,
                username: "dimka",
                avatar: PinzDomainAsset.userPlacholder.image
            )
        ]
    }
}

