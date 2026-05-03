import SwiftUI
import PinzUI
import PinzDomain
import PinzPins
import PinzBase

public struct AddMediaGroupingView: View {

    @State private var viewModel: AddMediaGroupingViewModel
    @State private var isMergePickerPresented = false
    @State private var showCancelConfirmation = false

    @Environment(\.appRouter) private var router

    public init(tripId: String, sessionId: String) {
        viewModel = AddMediaGroupingViewModel(tripId: tripId, sessionId: sessionId)
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
            } else {
                gradientWithButtons
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
        .mergePinsSheet(isPresented: $isMergePickerPresented, pins: viewModel.rawPins.pins) { first, second in
            viewModel.dispatch(.mergePins(firstIndex: first, secondIndex: second))
        }
        .confirmationDialog(
            "Отменить загрузку?",
            isPresented: $showCancelConfirmation,
            titleVisibility: .visible
        ) {
            Button("Отменить загрузку", role: .destructive) {
                Task { try? await viewModel.asyncDispatch(.cancel) }
            }
            Button("Продолжить", role: .cancel) {}
        }
    }

    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { showCancelConfirmation = true }
            )
        }, centerView: {
            HeaderTitle("Группировка медиа")
        }, rightView: {
            EmptyView()
        })
    }

    private var content: some View {
        let pins = viewModel.rawPins.pins
        return VStack(spacing: 0) {
            ForEach(pins.indices, id: \.self) { index in
                RawPinView(
                    pin: pins[index],
                    index: index,
                    allPins: pins,
                    onDeleteMedia: { media in
                        viewModel.dispatch(.deleteMedia(media, fromPin: pins[index].id))
                    },
                    onMoveMedia: { media, targetIndex in
                        viewModel.dispatch(.moveMedia(media, fromPin: index, toPin: targetIndex))
                    }
                )
                .padding(.horizontal, 12)
                if index != pins.count - 1 {
                    Divider().padding(.leading, 12)
                }
            }
        }
        .padding(.bottom, 200)
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons(height: 190) {
            VStack(spacing: 6) {
                HStack(spacing: 6) {
                    PinzButton(
                        type: .slot(style: .secondary(needBorder: true), title: "Добавить ещё"),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        action: .async { try await viewModel.asyncDispatch(.addMore) }
                    )
                    PinzButton(
                        type: .slot(style: .secondary(needBorder: true), title: "Объединить"),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        disabled: viewModel.rawPins.pins.count < 2,
                        action: .plain { isMergePickerPresented = true }
                    )
                    PinzButton(
                        type: .slot(style: .secondary(needBorder: true), title: "Добавить пин"),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        action: .plain { viewModel.dispatch(.addPin) }
                    )
                }
                PinzButton(
                    type: .slot(style: .primary, title: "Применить"),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .async { try await viewModel.asyncDispatch(.apply) }
                )
            }
        }
    }
}
