import SwiftUI

public extension View {
    func pinzA11y(_ element: PinzElement) -> some View {
        self.accessibilityIdentifier(element.accessibilityID)
    }
}
