import SwiftUI
import PinzUI
import PinzBase
import PinzNetworking

public struct PinUploadProcessingView: View {

    let tripId: String
    let sessionId: String

    @Environment(\.appRouter) private var router
    @State private var wsClient = PinUploadWebSocketClient()
    @State private var wsTask: Task<Void, Never>?
    @State private var pollingTask: Task<Void, Never>?
    @State private var navigated = false
    @State private var isCancelling = false
    @State private var showCancelConfirmation = false

    public init(tripId: String, sessionId: String) {
        self.tripId = tripId
        self.sessionId = sessionId
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                header
            } content: {
                EmptyView()
            }

            LoadingView(status: "Обрабатываем медиа...")
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .navigationBarBackButtonHidden(true)
        .onAppear {
            startWS()
            startPolling()
        }
        .onDisappear {
            teardown()
        }
        .confirmationDialog(
            "Отменить создание пина?",
            isPresented: $showCancelConfirmation,
            titleVisibility: .visible
        ) {
            Button("Отменить создание пина", role: .destructive) {
                Task { await cancelTapped() }
            }
            Button("Продолжить", role: .cancel) {}
        }
    }

    @ViewBuilder
    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { showCancelConfirmation = true }
            )
        }, centerView: {
            HeaderTitle("Создание пина")
        }, rightView: {
            EmptyView()
        })
    }

    private func startWS() {
        wsTask?.cancel()
        wsTask = Task {
            for await event in wsClient.connect(tripId: tripId, sessionId: sessionId) {
                if case let .processingCompleted(_, _, status) = event,
                   status.uppercased() == "READY_FOR_REVIEW" {
                    await navigateIfNeeded()
                    return
                }
            }
        }
    }

    private func startPolling() {
        pollingTask?.cancel()
        pollingTask = Task { @MainActor in
            while !Task.isCancelled && !navigated {
                try? await Task.sleep(for: .seconds(2))
                guard !Task.isCancelled else { return }
                guard !navigated else { return }
                let response = try? await NetworkService.shared.pinUploadGetReview(
                    tripId: tripId,
                    sessionId: sessionId
                )
                if response?.processingStatus == "READY_FOR_REVIEW" {
                    navigateIfNeeded()
                    return
                }
            }
        }
    }

    @MainActor
    private func navigateIfNeeded() {
        guard !navigated else { return }
        navigated = true
        teardown()
        router?.navigateToPinUploadReview(tripId: tripId, sessionId: sessionId)
    }

    @MainActor
    private func cancelTapped() async {
        guard !isCancelling else { return }
        isCancelling = true
        defer { isCancelling = false }

        do {
            try await NetworkService.shared.pinUploadCancel(tripId: tripId, sessionId: sessionId)
        } catch {
            // 409 / 404 — сессия уже закрыта или не найдена; storage всё равно чистим.
        }

        PinUploadSessionStorage.shared.clear(forTripId: tripId)
        teardown()
        router?.popToRoot()
    }

    private func teardown() {
        wsTask?.cancel()
        pollingTask?.cancel()
        wsClient.disconnect()
    }
}
