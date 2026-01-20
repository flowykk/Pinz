import SwiftUI

struct PinzSheetModifier: ViewModifier {
    func body(content: Content) -> some View {
        content
            .presentationCornerRadius(40)
            .presentationDragIndicator(.hidden)
    }
}

extension View {
    public func pinzSheet() -> some View {
        self.modifier(PinzSheetModifier())
    }
}
