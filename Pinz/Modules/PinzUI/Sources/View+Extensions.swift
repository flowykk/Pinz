import SwiftUI
import MapKit

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

    public func disabledWithOpacity(_ trueFlag: Bool) -> some View {
        self.disabled(trueFlag).opacity(trueFlag ? 0.7 : 1)
    }

    public func frame(_ size: CGFloat) -> some View {
        self.frame(width: size, height: size)
    }
    
    public func savedMapStyle(_ rawValue: String) -> some View {
        let mapStyle = (PinzMapStyle(rawValue: rawValue) ?? .satelight).toMapKitMapStyle()
        return self.mapStyle(mapStyle)
    }

    public func animationsDisabled() -> some View {
        self.transaction { transaction in
            transaction.animation = nil
        }
    }
}
