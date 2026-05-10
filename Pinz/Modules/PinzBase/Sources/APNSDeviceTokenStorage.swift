import Foundation

public final class APNSDeviceTokenStorage {
    public static let shared = APNSDeviceTokenStorage()

    public var hexToken: String?

    private init() {}
}
