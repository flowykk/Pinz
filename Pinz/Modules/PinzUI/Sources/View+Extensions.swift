import SwiftUI

extension View {
    @ViewBuilder
    public func `if`<Content: View>(_ condition: Bool, apply: (Self) -> Content) -> some View {
        if condition {
            apply(self)
        } else {
            self
        }
    }

    @ViewBuilder
    func ifLet<T, Content: View>(_ value: T?, apply: (Self, T) -> Content) -> some View {
        if let unwrapped = value {
            apply(self, unwrapped)
        } else {
            self
        }
    }

    public func frame(_ size: CGFloat) -> some View {
        self.frame(width: size, height: size)
    }
}
