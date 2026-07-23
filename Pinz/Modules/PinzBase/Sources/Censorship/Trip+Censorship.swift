import Foundation
import PinzDomain

public extension Trip {

    func displayName(in context: CensorshipContext) -> String {
        guard context == .public, isNameCensored else { return name }
        return CensorshipStubs.stub(for: .tripName, entityId: id)
    }

    func displayDescription(in context: CensorshipContext) -> String? {
        guard let description else { return nil }
        guard context == .public, isDescriptionCensored else { return description }
        return CensorshipStubs.stub(for: .tripDescription, entityId: id)
    }

    func censored(in context: CensorshipContext) -> Trip {
        guard context == .public else { return self }
        var copy = self
        copy.name = displayName(in: context)
        copy.description = displayDescription(in: context)
        return copy
    }
}
