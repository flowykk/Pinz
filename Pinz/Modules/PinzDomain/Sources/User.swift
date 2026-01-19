import Foundation
import SwiftUI

public struct User: Hashable, Identifiable {
    public let id: UUID = UUID()
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
