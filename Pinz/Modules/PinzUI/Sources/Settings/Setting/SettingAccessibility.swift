import SwiftUI
import PinzAccessibility

extension View {
    func applyPinzAccessibility(_ element: PinzElement?) -> some View {
        if let element {
            return AnyView(self.pinzA11y(element))
        }
        return AnyView(self)
    }
}
