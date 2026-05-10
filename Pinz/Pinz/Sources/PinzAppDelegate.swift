import UIKit
import UserNotifications

import PinzBase
import PinzNetworking

final class PinzAppDelegate: NSObject, UIApplicationDelegate, UNUserNotificationCenterDelegate {

    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]? = nil
    ) -> Bool {
        UNUserNotificationCenter.current().delegate = self
        Task {
            await Self.requestPushAuthorizationAndRegister(application: application)
        }
        return true
    }

    private static func requestPushAuthorizationAndRegister(application: UIApplication) async {
        let center = UNUserNotificationCenter.current()
        do {
            let granted = try await center.requestAuthorization(options: [.alert, .badge, .sound])
            guard granted else { return }
            await MainActor.run {
                application.registerForRemoteNotifications()
            }
        } catch {
            print("[APNS] requestAuthorization failed: \(error)")
        }
    }

    func application(_ application: UIApplication, didRegisterForRemoteNotificationsWithDeviceToken deviceToken: Data) {
        let hex = deviceToken.map { String(format: "%02.2hhx", $0) }.joined()
        APNSDeviceTokenStorage.shared.hexToken = hex
        Task {
            await PushNotificationRegistration.syncRegisteredTokenWithBackendIfPossible()
        }
    }

    func application(_ application: UIApplication, didFailToRegisterForRemoteNotificationsWithError error: Error) {
        print("[APNS] didFailToRegisterForRemoteNotifications: \(error)")
    }

    // MARK: - UNUserNotificationCenterDelegate

    func userNotificationCenter(
        _: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        completionHandler([.banner, .list, .sound])
    }

    func userNotificationCenter(
        _: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        let userInfo = response.notification.request.content.userInfo
        Task {
            await PushTripFromNotificationCoordinator.shared.handle(userInfo: userInfo)
            completionHandler()
        }
    }
}
