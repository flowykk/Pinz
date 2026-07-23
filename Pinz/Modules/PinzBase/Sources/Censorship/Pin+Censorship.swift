import Foundation
import PinzDomain

public extension Pin {

    private var censorshipEntityId: String {
        serverId ?? id
    }

    func displayName(in context: CensorshipContext) -> String {
        guard context == .public, isNameCensored else { return name }
        return CensorshipStubs.stub(for: .pinName, entityId: censorshipEntityId)
    }

    func displayDescription(in context: CensorshipContext) -> String? {
        guard let description else { return nil }
        guard context == .public, isDescriptionCensored else { return description }
        return CensorshipStubs.stub(for: .pinDescription, entityId: censorshipEntityId)
    }

    func censored(in context: CensorshipContext) -> Pin {
        guard context == .public else { return self }
        var copy = self
        copy.name = displayName(in: context)
        copy.description = displayDescription(in: context)
        return copy
    }
}
