import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

public struct AddMediaReviewView: View {

    @State private var viewModel: AddMediaReviewViewModel
    @State private var now = Date()
    @State private var showCancelConfirmation = false

    private let timer = Timer.publish(every: 1, on: .main, in: .common).autoconnect()

    @Environment(\.appRouter) private var router

    public init(tripId: String, sessionId: String) {
        viewModel = AddMediaReviewViewModel(tripId: tripId, sessionId: sessionId)
    }

    private var canTakeover: Bool {
        guard let takeoverAt = viewModel.takeoverAvailableAt else { return true }
        return now >= takeoverAt
    }

    private var timeUntilTakeover: String {
        guard let takeoverAt = viewModel.takeoverAvailableAt, !canTakeover else { return "" }
        let seconds = max(0, Int(takeoverAt.timeIntervalSince(now)))
        return "\(seconds / 60):\(String(format: "%02d", seconds % 60))"
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                header
            } content: {
                if !viewModel.isLoading {
                    content
                }
            }

            if viewModel.isLoading {
                LoadingView()
            } else if viewModel.canEdit {
                editorButtons
            } else {
                takeoverBanner
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
        .onReceive(timer) { _ in now = Date() }
        .confirmationDialog(
            PinzBaseStrings.AddMedia.Cancel.title,
            isPresented: $showCancelConfirmation,
            titleVisibility: .visible
        ) {
            Button(PinzBaseStrings.AddMedia.Cancel.confirm, role: .destructive) {
                Task { try? await viewModel.asyncDispatch(.cancel) }
            }
            Button(PinzBaseStrings.PinUpload.Dialog.continue, role: .cancel) {}
        }
    }

    @ViewBuilder
    private var header: some View {
        Header(leftView: {
            if viewModel.canEdit {
                PinzButton(
                    type: .icon(.chevronLeft),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { showCancelConfirmation = true }
                )
            } else {
                EmptyView()
            }
        }, centerView: {
            HeaderTitle(PinzBaseStrings.AddMedia.Review.title)
        }, rightView: {
            EmptyView()
        })
    }

    private var content: some View {
        let pins = viewModel.pins
        return VStack(spacing: 8) {
            ForEach(pins.indices, id: \.self) { index in
                ReviewPinView(pin: pins[index], index: index) {
                    if viewModel.canEdit {
                        viewModel.navigateToPinInfo(at: index)
                    }
                }
                if index != pins.count - 1 {
                    Divider().padding(.leading, 12)
                }
            }
        }
        .padding(.horizontal, 12)
        .padding(.bottom, 100)
        .animation(.default, value: viewModel.pins)
    }

    private var editorButtons: some View {
        BottomGradientWithButtons {
            HStack(spacing: 8) {
                PinzButton(
                    type: .slot(style: .secondary(needBorder: true), title: PinzBaseStrings.Common.Button.cancel),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .plain { showCancelConfirmation = true }
                )
                PinzButton(
                    type: .slot(style: .primary, title: PinzBaseStrings.Common.Button.confirm),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .async { try await viewModel.asyncDispatch(.confirm) }
                )
            }
        }
    }

    private var takeoverBanner: some View {
        BottomGradientWithButtons {
            VStack(spacing: 8) {
                if let initiator = viewModel.currentInitiator {
                    let takeoverText: String = timeUntilTakeover.isEmpty
                        ? PinzBaseStrings.AddMedia.Review.Takeover.message(initiator.username)
                        : PinzBaseStrings.AddMedia.Review.Takeover.messageWithCountdown(initiator.username, timeUntilTakeover)
                    Text(takeoverText)
                        .roundedFont(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                        .multilineTextAlignment(.center)
                }
                PinzButton(
                    type: .slot(style: .primary, title: PinzBaseStrings.AddMedia.Button.takeover),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .async { try await viewModel.asyncDispatch(.takeover) }
                )
                .disabledWithOpacity(!canTakeover)
            }
        }
    }
}
