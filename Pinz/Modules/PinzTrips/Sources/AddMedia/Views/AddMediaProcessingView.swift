import SwiftUI
import PinzUI
import PinzBase
import PinzNetworking

public struct AddMediaProcessingView: View {

    let tripId: String
    let sessionId: String

    @Environment(\.appRouter) private var router
    @State private var wsClient = AddMediaWebSocketClient()
    @State private var wsTask: Task<Void, Never>?

    public init(tripId: String, sessionId: String) {
        self.tripId = tripId
        self.sessionId = sessionId
    }

    public var body: some View {
        LoadingView(status: "Обрабатываем медиа...")
            .background(PinzUIAsset.background.swiftUIColor)
            .navigationBarBackButtonHidden(true)
            .onAppear { startListening() }
            .onDisappear {
                wsTask?.cancel()
                wsClient.disconnect()
            }
    }

    private func startListening() {
        wsTask?.cancel()
        wsTask = Task {
            for await event in wsClient.connect(tripId: tripId) {
                if case let .tripStatusChanged(status) = event,
                   status.uppercased() == "ADD_MEDIA_DRAFT_FINAL_REVIEW" {
                    wsClient.disconnect()
                    router?.navigateToAddMediaReview(tripId: tripId, sessionId: sessionId)
                    return
                }
            }
        }
    }
}
