import SwiftUI
import PhotosUI
import PinzUI
import PinzDomain
import PinzBase

public struct PinUploadStartView: View {

    @State private var viewModel: PinUploadStartViewModel
    @State private var isMediaPickerPresented = false
    @State private var pickerItems: [PhotosPickerItem] = []

    let galleryColumns: Int = 3
    let gallerySpacing: CGFloat = 4

    @Environment(\.appRouter) private var router

    public init(tripId: String, targetPinId: String? = nil) {
        viewModel = PinUploadStartViewModel(tripId: tripId, targetPinId: targetPinId)
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                header
            } content: {
                if !viewModel.isLoading {
                    gallery
                }
            }

            if viewModel.isLoading {
                LoadingView(status: viewModel.loadingStatus?.localizedValue)
            } else {
                gradientWithButtons
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
        .photosPicker(
            isPresented: $isMediaPickerPresented,
            selection: $pickerItems,
            maxSelectionCount: nil,
            matching: .any(of: [.images, .videos])
        )
        .onChange(of: pickerItems) { _, newItems in
            guard !newItems.isEmpty else { return }
            viewModel.dispatch(.addMedias(newItems))
            pickerItems = []
        }
    }

    @ViewBuilder
    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { viewModel.dispatch(.navigate(.back)) }
            )
        }, centerView: {
            HeaderTitle(
                viewModel.targetPinId != nil
                    ? PinzBaseStrings.PinUpload.Header.addMedia
                    : PinzBaseStrings.PinUpload.Header.createPin
            )
        }, rightView: {
            PinzButton(
                type: .icon(.plus),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { isMediaPickerPresented = true }
            )
        })
    }

    private var gallery: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: gallerySpacing) {
                PinzGrid($viewModel.medias, columns: galleryColumns, spacing: gallerySpacing) { media, _ in
                    LoadedMediaThumbnailView(
                        media: media,
                        contentMode: .fit,
                        cornerRadius: 14,
                        onMediaDelete: {
                            withAnimation(.easeInOut(duration: 0.3)) {
                                viewModel.dispatch(.deleteMedia(media.id))
                            }
                        }
                    )
                }
            }
        }
        .scrollIndicators(.hidden)
        .padding(.horizontal, gallerySpacing)
        .padding(.bottom, 60)
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            PinzButton(
                type: .slot(style: .primary, title: PinzBaseStrings.Common.Button.next),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                action: .async { try await viewModel.asyncDispatch(.start) }
            )
            .accessibilityIdentifier("pinUpload.button.next")
            .disabledWithOpacity(viewModel.medias.isEmpty)
        }
    }
}
