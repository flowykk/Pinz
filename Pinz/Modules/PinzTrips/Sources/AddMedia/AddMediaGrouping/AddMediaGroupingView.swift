import SwiftUI
import PinzBase
import PinzUI
import PinzDomain
import PinzPins

struct AddMediaGroupingView: View {
    @State private var viewModel: AddMediaGroupingViewModel

    private let onBack: () -> Void
    private let onRetry: () async -> Void
    private let onContinue: () -> Void

    init(
        viewModel: AddMediaGroupingViewModel,
        onBack: @escaping () -> Void,
        onRetry: @escaping () async -> Void,
        onContinue: @escaping () -> Void
    ) {
        self._viewModel = State(wrappedValue: viewModel)
        self.onBack = onBack
        self.onRetry = onRetry
        self.onContinue = onContinue
    }

    var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                Header(
                    leftView: {
                        PinzButton(
                            type: .icon(.chevronLeft),
                            tint: PinzUIAsset.textPrimary.swiftUIColor,
                            action: .plain { onBack() }
                        )
                    },
                    centerView: {
                        HeaderTitle(PinzBaseStrings.TripPins.Button.addMedia)
                    }
                )
            } content: {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        if !viewModel.existingPinsPreview.isEmpty {
                            sectionTitle(PinzBaseStrings.AddMedia.Label.currentTripPins)
                            renderPins(viewModel.existingPinsPreview, allowEdits: false)
                        }

                        if !viewModel.draftPins.isEmpty {
                            sectionTitle(PinzBaseStrings.AddMedia.Label.draftGroups)
                            renderPins(viewModel.draftPins, allowEdits: true)
                        }

                        if !viewModel.isLoading && viewModel.draftPins.isEmpty && !viewModel.hasFailed {
                            Text(PinzBaseStrings.AddMedia.Message.noGroupsGeneratedYet)
                                .roundedFont(size: 14)
                                .foregroundStyle(.secondary)
                                .padding(.horizontal, 12)
                        }

                        if !viewModel.isLoading && !viewModel.isReady && !viewModel.hasFailed {
                            Text(PinzBaseStrings.AddMedia.Message.waitingForGrouping)
                                .roundedFont(size: 14)
                                .foregroundStyle(.secondary)
                                .padding(.horizontal, 12)
                        }
                    }
                    .padding(.top, 12)
                    .padding(.bottom, 170)
                }
                .scrollIndicators(.hidden)
            }

            if viewModel.isLoading {
                LoadingView(status: viewModel.loadingStatus?.localizedValue)
            }

            gradientWithButtons
        }
        .background(PinzUIAsset.background.swiftUIColor)
    }

    private func sectionTitle(_ title: String) -> some View {
        Text(title)
            .roundedFont(size: 16, weight: .semibold)
            .padding(.horizontal, 12)
    }

    @ViewBuilder
    private func renderPins(_ pins: [RawPin], allowEdits: Bool) -> some View {
        VStack(spacing: 8) {
            ForEach(pins.indices, id: \.self) { index in
                RawPinView(
                    pin: pins[index],
                    index: index,
                    allPins: pins,
                    onDeleteMedia: allowEdits ? { media in
                        viewModel.dispatch(.deleteMedia(media, fromPin: pins[index].id))
                    } : nil,
                    onMoveMedia: allowEdits ? { media, targetIndex in
                        viewModel.dispatch(.moveMedia(media, fromPin: index, toPin: targetIndex))
                    } : nil,
                    isMediaLocked: { media in
                        if !allowEdits {
                            return true
                        }
                        return viewModel.existingMediaIds.contains(media.id)
                    }
                )
                .padding(.horizontal, 12)

                if index != pins.count - 1 {
                    Divider()
                        .padding(.leading, 12)
                }
            }
        }
    }

    private var gradientWithButtons: some View {
        Group {
            if viewModel.isLoading {
                EmptyView()
            } else if viewModel.hasFailed {
                BottomGradientWithButtons {
                    PinzButton(
                        type: .slot(style: .secondary(needBorder: true), title: PinzBaseStrings.Common.Button.retry),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        action: .async { await onRetry() }
                    )
                }
            } else {
                BottomGradientWithButtons {
                    PinzButton(
                        type: .slot(style: .primary, title: PinzBaseStrings.Common.Button.next),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        disabled: !viewModel.canProceed,
                        action: .plain { onContinue() }
                    )
                }
            }
        }
    }
}
