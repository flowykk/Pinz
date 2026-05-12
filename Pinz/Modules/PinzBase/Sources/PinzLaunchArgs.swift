import Foundation

public enum PinzLaunchArg {
    public static let useLocalhost = "-useLocalhost"
    public static let networkStub = "-networkStub"
    public static let fakeTokens = "-fakeTokens"
    public static let testingProfile = "-testingProfile"
    public static let testingWishlist = "-testingWishlist"
    public static let testingTrip = "-testingTrip"
    public static let testingTripId = "-testingTripId"
    public static let testingPinUploadFakeMedia = "-testingPinUploadFakeMedia"
    public static let testingTripCreation = "-testingTripCreation"
    public static let testingTripCreationFakeMedia = "-testingTripCreationFakeMedia"
}

public enum PinzLaunchArgs {
    public static let fakeAccessToken = "fake_access_token"
    public static let fakeRefreshToken = "fake_refresh_token"

    @discardableResult
    public static func apply() -> Bool {
        guard CommandLine.arguments.contains(PinzLaunchArg.fakeTokens) else {
            return false
        }

        TokenStorage.shared.save(
            accessToken: fakeAccessToken,
            refreshToken: fakeRefreshToken
        )

        if testingTrip, let tripId = testingTripIdValue {
            SelectedTripStorage.shared.selectTrip(id: tripId)
        } else {
            SelectedTripStorage.shared.clearSelection()
        }
        return true
    }

    public static var hasFakeTokens: Bool {
        CommandLine.arguments.contains(PinzLaunchArg.fakeTokens)
    }

    public static var useLocalhost: Bool {
        CommandLine.arguments.contains(PinzLaunchArg.useLocalhost)
    }

    public static var useNetworkStub: Bool {
        CommandLine.arguments.contains(PinzLaunchArg.networkStub)
    }

    public static var baseURL: String {
        useLocalhost ? "http://localhost:8080" : "https://pinz.website"
    }

    public static var websocketURLString: String {
        useLocalhost ? "ws://localhost:8080" : "wss://pinz.website"
    }

    public static var testingProfile: Bool {
        CommandLine.arguments.contains(PinzLaunchArg.testingProfile)
    }

    public static var testingWishlist: Bool {
        CommandLine.arguments.contains(PinzLaunchArg.testingWishlist)
    }

    public static var testingTrip: Bool {
        CommandLine.arguments.contains(PinzLaunchArg.testingTrip)
    }

    public static var testingPinUploadFakeMedia: Bool {
        CommandLine.arguments.contains(PinzLaunchArg.testingPinUploadFakeMedia)
    }

    public static var testingTripCreation: Bool {
        CommandLine.arguments.contains(PinzLaunchArg.testingTripCreation)
    }

    public static var testingTripCreationFakeMedia: Bool {
        CommandLine.arguments.contains(PinzLaunchArg.testingTripCreationFakeMedia)
    }

    public static var testingTripIdValue: String? {
        guard let index = CommandLine.arguments.firstIndex(of: PinzLaunchArg.testingTripId),
              CommandLine.arguments.count > index + 1 else {
            return nil
        }
        return CommandLine.arguments[index + 1]
    }
}
