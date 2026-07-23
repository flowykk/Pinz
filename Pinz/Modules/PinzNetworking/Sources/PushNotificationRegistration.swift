import Foundation
import PinzBase

public enum PushNotificationRegistration {

    public static func syncRegisteredTokenWithBackendIfPossible() async {
        guard TokenStorage.shared.isAuthenticated,
              let hex = APNSDeviceTokenStorage.shared.hexToken else { return }
        do {
            _ = try await NetworkService.shared.registerDeviceToken(apnsToken: hex)
        } catch {
            print("[APNS] registerDeviceToken failed: \(error)")
        }
    }

    public static func unregisterFromBackendIfPossible() async {
        guard TokenStorage.shared.isAuthenticated,
              let hex = APNSDeviceTokenStorage.shared.hexToken else { return }
        do {
            _ = try await NetworkService.shared.unregisterDeviceToken(apnsToken: hex)
        } catch {
            print("[APNS] unregisterDeviceToken failed: \(error)")
        }
    }
}
