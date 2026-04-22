import SwiftUI
import PinzBase
import PinzUI
import PinzDomain
import PinzPins

struct AddMediaGroupingView: View {
    @State private var viewModel: AddMediaGroupingViewModel

    private let onBack: () -> Void
    private let onRetry: () async -> Void
    private let onContinue: () async -> Void

    init(
        viewModel: AddMediaGroupingViewModel,
        onBack: @escaping () -> Void,
        onRetry: @escaping () async -> Void,
        onContinue: @escaping () async -> Void
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
                        HeaderTitle(
                            PinzBaseStrings.TripPins.Button.addMedia,
                            subtitle: PinzBaseStrings.AddMedia.Subtitle.step1
                        )
                    }
                )
            } content: {
                ScrollView {
                    VStack(alignment: .leading, spacing: 16) {
                        if !allPins.isEmpty {
                            renderPins(allPins)
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

            if viewModel.isLoading && !isApplying {
                LoadingView(status: viewModel.loadingStatus?.localizedValue)
            }

            gradientWithButtons
        }
        .background(PinzUIAsset.background.swiftUIColor)
    }

    private var allPins: [RawPin] {
        dedupePinsByMediaId(viewModel.existingPinsPreview + viewModel.draftPins)
    }

    private var draftPinIds: Set<String> {
        Set(viewModel.draftPins.map(\.id))
    }

    private var isApplying: Bool {
        viewModel.loadingStatus == .applying
    }

    @ViewBuilder
    private func renderPins(_ pins: [RawPin]) -> some View {
        VStack(spacing: 8) {
            ForEach(pins.indices, id: \.self) { index in
                let sourcePin = pins[index]
                RawPinView(
                    pin: sourcePin,
                    index: index,
                    allPins: pins,
                    movablePinIds: draftPinIds,
                    onDeleteMedia: { media in
                        guard draftPinIds.contains(sourcePin.id),
                              !viewModel.existingMediaIds.contains(media.id) else {
                            return
                        }
                        viewModel.dispatch(.deleteMedia(media, fromPin: sourcePin.id))
                    },
                    onMoveMedia: { media, targetIndex in
                        guard draftPinIds.contains(sourcePin.id),
                              !viewModel.existingMediaIds.contains(media.id),
                              targetIndex >= 0,
                            targetIndex < pins.count,
                              draftPinIds.contains(pins[targetIndex].id) else {
                            return
                        }
                        viewModel.dispatch(
                            .moveMedia(
                                media,
                                fromPinId: sourcePin.id,
                                toPinId: pins[targetIndex].id
                            )
                        )
                    },
                    isMediaLocked: { media in
                        viewModel.existingMediaIds.contains(media.id)
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

    private func dedupePinsByMediaId(_ pins: [RawPin]) -> [RawPin] {
        var seenMediaIds = Set<String>()

        return pins.compactMap { pin in
            var updatedPin = pin
            updatedPin.medias.removeAll { media in
                if seenMediaIds.contains(media.id) {
                    return true
                }
                seenMediaIds.insert(media.id)
                return false
            }
            return updatedPin.medias.isEmpty ? nil : updatedPin
        }
    }

    private var gradientWithButtons: some View {
        Group {
            if viewModel.isLoading && !isApplying {
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
                        action: .async { await onContinue() }
                    )
                }
            }
        }
    }
}
