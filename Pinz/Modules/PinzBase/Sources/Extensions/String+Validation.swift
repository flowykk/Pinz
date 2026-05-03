import Foundation

public extension String {
    var isValidEmail: Bool {
        range(of: #"^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$"#, options: .regularExpression) != nil
    }
}
