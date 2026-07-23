import Foundation

/// Where draft pin state lives while resolving `Pin.Issue` flows.
public enum PinProblemsDraftBinding: Hashable {
    case tripCreation(tripId: String)
    case pinUpload(sessionId: String)
    case addMediaReview(sessionId: String)
}
