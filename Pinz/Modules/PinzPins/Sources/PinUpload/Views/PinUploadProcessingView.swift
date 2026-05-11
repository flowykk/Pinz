import SwiftUI
import PinzUI
import PinzBase
import PinzNetworking

public struct PinUploadProcessingView: View {

    let tripId: String
    let sessionId: String
    let targetPinId: String?

    @Environment(\.appRouter) private var router
    @State private var wsClient = PinUploadWebSocketClient()
    @State private var wsTask: Task<Void, Never>?
    @State private var pollingTask: Task<Void, Never>?
    @State private var navigated = false
    @State private var isCancelling = false
    @State private var showCancelConfirmation = false

    private var isAdditionFlow: Bool { targetPinId != nil }

    private var headerTitle: String {
        isAdditionFlow ? PinzBaseStrings.PinUpload.Header.addMedia : PinzBaseStrings.PinUpload.Header.createPin
    }

    public init(tripId: String, sessionId: String, targetPinId: String? = nil) {
        self.tripId = tripId
        self.sessionId = sessionId
        self.targetPinId = targetPinId
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                header
            } content: {
                EmptyView()
            }

            LoadingView(status: PinzBaseStrings.PinUpload.Processing.status)
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
            isAdditionFlow
                ? PinzBaseStrings.PinUpload.Cancel.Addition.title
                : PinzBaseStrings.PinUpload.Cancel.Creation.title,
            isPresented: $showCancelConfirmation,
            titleVisibility: .visible
        ) {
            Button(
                isAdditionFlow
                    ? PinzBaseStrings.PinUpload.Cancel.Addition.confirm
                    : PinzBaseStrings.PinUpload.Cancel.Creation.confirm,
                role: .destructive
            ) {
                Task { await cancelTapped() }
            }
            Button(PinzBaseStrings.PinUpload.Dialog.continue, role: .cancel) {}
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
            HeaderTitle(headerTitle)
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
        router?.navigateToPinUploadReview(tripId: tripId, sessionId: sessionId, targetPinId: targetPinId)
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

        if let pinId = targetPinId {
            PinUploadAdditionSessionStorage.shared.clear(tripId: tripId, pinId: pinId)
            teardown()
            router?.popAllPinUploadRoutes()
        } else {
            PinUploadSessionStorage.shared.clear(forTripId: tripId)
            teardown()
            router?.popToRoot()
        }
    }

    private func teardown() {
        wsTask?.cancel()
        pollingTask?.cancel()
        wsClient.disconnect()
    }
}
