import SwiftUI
import PhotosUI
import PinzBase
import PinzDomain
import PinzUI

struct AddMediaSelectionView: View {
    @State private var viewModel: AddMediaSelectionViewModel

    @State private var pickerItems: [PhotosPickerItem] = []
    @State private var isPickerPresented = true

    private let onBack: () -> Void
    private let onSessionReady: (_ session: AddMediaStartDTO, _ loadedMedia: [LoadedMedia]) -> Void

    init(
        viewModel: AddMediaSelectionViewModel,
        onBack: @escaping () -> Void,
        onSessionReady: @escaping (_ session: AddMediaStartDTO, _ loadedMedia: [LoadedMedia]) -> Void
    ) {
        self._viewModel = State(wrappedValue: viewModel)
        self.onBack = onBack
        self.onSessionReady = onSessionReady
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                Header(leftView: {
                    PinzButton(
                        type: .icon(.chevronLeft),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { onBack() }
                    )
                }, centerView: {
                    HeaderTitle(PinzBaseStrings.TripPins.Button.addMedia)
                })
            } content: {
                VStack(spacing: 16) {
                    if let session = viewModel.session {
                        VStack(alignment: .leading, spacing: 8) {
                            Text(PinzBaseStrings.AddMedia.Label.sessionStarted)
                                .roundedFont(size: 18, weight: .semibold)
                            Text(PinzBaseStrings.AddMedia.Label.sessionIdPrefix + session.sessionId)
                                .roundedFont(size: 14, weight: .regular)
                                .foregroundStyle(.secondary)
                            Text(PinzBaseStrings.AddMedia.Label.statusPrefix + session.status)
                                .roundedFont(size: 14, weight: .regular)
                                .foregroundStyle(.secondary)
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal, 12)
                    } else {
                        VStack(spacing: 8) {
                            Text(PinzBaseStrings.AddMedia.Hint.chooseMedia)
                                .roundedFont(size: 16, weight: .medium)
                            Text(PinzBaseStrings.AddMedia.Description.sessionHint)
                                .roundedFont(size: 14)
                                .multilineTextAlignment(.center)
                                .foregroundColor(.secondary)
                        }
                        .padding(.horizontal, 12)
                    }

                }
                .padding(.bottom, 130)
                .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .center)
            }

            if viewModel.isLoading {
                LoadingView(status: viewModel.loadingStatus?.localizedValue)
            }

            gradientWithButtons
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .photosPicker(
            isPresented: $isPickerPresented,
            selection: $pickerItems,
            maxSelectionCount: nil,
            matching: .any(of: [.images, .videos])
        )
        .onAppear {
            isPickerPresented = true
        }
        .onChange(of: pickerItems) { _, newItems in
            guard !newItems.isEmpty else { return }
            Task {
                await viewModel.asyncDispatch(.startSession(items: newItems))
                if let session = viewModel.session {
                    onSessionReady(session, viewModel.loadedMedia)
                }
                pickerItems = []
            }
        }
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            PinzButton(
                type: .slot(style: .secondary(needBorder: true), title: PinzBaseStrings.TripPins.Button.addMedia),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                action: .plain { isPickerPresented = true }
            )
        }
    }
}
