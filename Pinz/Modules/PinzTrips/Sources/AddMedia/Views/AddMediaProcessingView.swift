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
    @State private var stubWsFallbackTask: Task<Void, Never>?
    @State private var reviewPollTask: Task<Void, Never>?
    @State private var didNavigateToReview = false

    public init(tripId: String, sessionId: String) {
        self.tripId = tripId
        self.sessionId = sessionId
    }

    public var body: some View {
        LoadingView(status: PinzBaseStrings.PinUpload.Processing.status)
            .background(PinzUIAsset.background.swiftUIColor)
            .navigationBarBackButtonHidden(true)
            .onAppear { startListening() }
            .onDisappear {
                wsTask?.cancel()
                stubWsFallbackTask?.cancel()
                reviewPollTask?.cancel()
                wsClient.disconnect()
            }
    }

    private func startListening() {
        wsTask?.cancel()
        stubWsFallbackTask?.cancel()
        reviewPollTask?.cancel()
        wsTask = Task {
            for await event in wsClient.connect(tripId: tripId) {
                if case let .tripStatusChanged(status) = event,
                   status.uppercased() == "ADD_MEDIA_DRAFT_FINAL_REVIEW" {
                    await MainActor.run { navigateToReviewFromProcessing() }
                    return
                }
            }
        }
        // `-networkStub`: HTTP стабится Moya, WebSocket — нет, событий не будет.
        if PinzLaunchArgs.useNetworkStub {
            stubWsFallbackTask = Task { @MainActor in
                try? await Task.sleep(for: .milliseconds(500))
                guard !Task.isCancelled else { return }
                navigateToReviewFromProcessing()
            }
        }
        reviewPollTask = Task { @MainActor in
            for attempt in 1...60 {
                try? await Task.sleep(for: .milliseconds(1500))
                guard !Task.isCancelled, !didNavigateToReview else { return }
                do {
                    _ = try await NetworkService.shared.addMediaGetReview(tripId: tripId, sessionId: sessionId)
                    navigateToReviewFromProcessing()
                    return
                } catch {
                    if attempt <= 2 || attempt % 5 == 0 {
                        print("[AddMediaProcessingView] poll getReview (\(attempt)/60): \(error.localizedDescription)")
                    }
                }
            }
            print("[AddMediaProcessingView] poll getReview timed out")
        }
    }

    private func navigateToReviewFromProcessing() {
        guard !didNavigateToReview else { return }
        didNavigateToReview = true
        wsTask?.cancel()
        stubWsFallbackTask?.cancel()
        reviewPollTask?.cancel()
        wsClient.disconnect()
        router?.navigateToAddMediaReview(tripId: tripId, sessionId: sessionId)
    }
}
