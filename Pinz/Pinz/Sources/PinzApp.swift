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
    @State private var router = AppRouter(
//        initialPath: TokenStorage.shared.isAuthenticated ? [.main] : []
        initialPath: {
            #if DEBUG
            let reviewRoute = Route.tripCreation(
                .final(
                    tripId: "mock-trip-id",
                    pins: Array(Pin.stubs().prefix(3))
                )
            )
            if TokenStorage.shared.isAuthenticated {
                return [.main, reviewRoute]
            }
            return [reviewRoute]
            #else
            return TokenStorage.shared.isAuthenticated ? [.main] : []
            #endif
        }()
    )

    var body: some Scene {
        WindowGroup {
            ZStack {
//                FeedView()
                RootView(router: router) {
                    AuthFlowView()
                }

#if DEBUG
                debugResetButton
#endif
            }
        }
    }

#if DEBUG
    @State private var showResetConfirmation = false

    private var debugResetButton: some View {
        VStack {
            Spacer()
            HStack {
                Button {
                    showResetConfirmation = true
                } label: {
                    Color.red.opacity(0.6)
                        .frame(width: 20, height: 20)
                        .clipShape(Rectangle())
                }
                .alert("Сбросить данные?", isPresented: $showResetConfirmation) {
                    Button("Сбросить", role: .destructive) {
                        TokenStorage.shared.clear()
                        UserDefaults.standard.removePersistentDomain(
                            forName: Bundle.main.bundleIdentifier ?? ""
                        )
                        router.path = []
                    }
                    Button("Отмена", role: .cancel) {}
                } message: {
                    Text("Удалит токены и UserDefaults. Приложение вернётся к экрану авторизации.")
                }
                Spacer()
            }
            Spacer()
        }
        .ignoresSafeArea()
        .allowsHitTesting(true)
    }
#endif
}
