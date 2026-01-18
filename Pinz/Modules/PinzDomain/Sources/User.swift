import SwiftUI

public struct User: Hashable {
    public var nickname: String
    public var email: String

    public init(
        nickname: String,
        email: String
    ) {
        self.nickname = nickname
        self.email = email
    }
}
