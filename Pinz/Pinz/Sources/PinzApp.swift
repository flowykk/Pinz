import SwiftUI
import PinzAuthentication
import PinzNavigation
import PinzTrips
import CoreLocation

import PinzBase
import PinzDomain
import PinzUI
import PinzFeed
import PinzPins

@main
struct PinzApp: App {
    @UIApplicationDelegateAdaptor(PinzAppDelegate.self) private var appDelegate

    @State private var router: AppRouter
    @State private var toastController = ToastController()

    init() {
        TokenStorage.shared.save(
            accessToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODM0MTkzNTksImlhdCI6MTc4MDgyNzM1OSwidXNlcl9pZCI6IjIxM2ExMjg3LTBkNDItNGM0ZS05YTlhLTBmNzhjMWRlZTg3MiIsInVzZXJuYW1lIjoiRGRkZHNkZnNkZnNkZiJ9.E36d8RFX1NCDxf8UxKy9qzMLGc04Tps7z2AEtFvZxUk",
            refreshToken: "DO9Oi63q+p7/XxqzyLeq6zU+DYws3YI2dd01MfBPcKQ="
        )
        PinzLaunchArgs.apply()
        _router = State(initialValue: AppRouter(initialPath: Self.initialRoutePath()))
    }

    private static func initialRoutePath() -> [Route] {
        if PinzLaunchArgs.testingTripCreation {
            return [.main, .tripCreation(.initial)]
        }
        return TokenStorage.shared.isAuthenticated ? [.main] : []
    }

    var body: some Scene {
        WindowGroup {
            ZStack {
                RootView(router: router) {
                    AuthFlowView()
                }
                .setupToast(toastController: toastController)
                .environment(\.showToast, {
                    toastController.present(with: $0)
                })
                .onOpenURL { url in
                    Task {
                        await TripInviteDeepLinkCoordinator(
                            router: router,
                            showToast: { toastController.present(with: $0) }
                        ).handleIncomingURL(url)
                    }
                }
                .onReceive(NotificationCenter.default.publisher(for: .pinzDidAuthenticate)) { _ in
                    Task {
                        await TripInviteDeepLinkCoordinator(
                            router: router,
                            showToast: { toastController.present(with: $0) }
                        ).processPendingInviteIfNeeded()
                    }
                }
                .onReceive(NotificationCenter.default.publisher(for: .pinzSessionInvalidated)) { _ in
                    router.navigateToAuthenticationRoot()
                }
                .task {
                    await TripInviteDeepLinkCoordinator(
                        router: router,
                        showToast: { toastController.present(with: $0) }
                    ).processPendingInviteIfNeeded()
                }
                .onAppear {
                    PushTripFromNotificationCoordinator.shared.router = router
                    PushTripFromNotificationCoordinator.shared.showToast = {
                        toastController.present(with: $0)
                    }
                }
            }
        }
    }
}

extension View {
    fileprivate func setupToast(toastController: ToastController) -> some View {
        self.overlay {
            ToastView(controller: toastController)
        }
    }
}
