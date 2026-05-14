import SwiftUI
import PhotosUI
import PinzUI
import PinzDomain
import PinzBase

public struct AddMediaUploadingView: View {

    @State private var viewModel: AddMediaUploadingViewModel
    @State private var isMorePickerPresented = false
    @State private var pickerItems: [PhotosPickerItem] = []
    @State private var showCancelConfirmation = false

    let galleryColumns: Int = 3
    let gallerySpacing: CGFloat = 4

    @Environment(\.appRouter) private var router

    public init(tripId: String, sessionId: String) {
        viewModel = AddMediaUploadingViewModel(tripId: tripId, sessionId: sessionId)
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                header
            } content: {
                gallery
            }

            if viewModel.isLoading {
                LoadingView()
            } else {
                gradientWithButtons
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
        .photosPicker(
            isPresented: $isMorePickerPresented,
            selection: $pickerItems,
            maxSelectionCount: nil,
            matching: .any(of: [.images, .videos])
        )
        .onChange(of: pickerItems) { _, newItems in
            guard !newItems.isEmpty else { return }
            Task { try? await viewModel.asyncDispatch(.addMore(newItems)) }
            pickerItems = []
        }
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
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { showCancelConfirmation = true }
            )
        }, centerView: {
            HeaderTitle(PinzBaseStrings.AddMedia.Uploading.title)
        }, rightView: {
            EmptyView()
        })
    }

    private var gallery: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: gallerySpacing) {
                PinzGrid($viewModel.uploadedMediaEntries, columns: galleryColumns, spacing: gallerySpacing) { entry, _ in
                    LoadableImageThumbnail(url: URL(string: entry.url)) { state in
                        mediaThumbnail(for: state)
                    }
                }
            }
        }
        .scrollIndicators(.hidden)
        .padding(.horizontal, gallerySpacing)
        .padding(.bottom, 60)
    }

    @ViewBuilder
    private func mediaThumbnail(for state: LoadableMediaState) -> some View {
        switch state {
        case .empty:
            Rectangle()
                .fill(Color.gray.opacity(0.3))
                .cornerRadius(14)
                .overlay { ProgressView().tint(.white) }
        case .ready(let image):
            Image(uiImage: image)
                .resizable()
                .scaledToFill()
                .cornerRadius(14)
                .clipped()
        case .failure:
            Rectangle()
                .fill(Color.gray.opacity(0.3))
                .cornerRadius(14)
                .overlay {
                    Image(systemName: "exclamationmark.triangle")
                        .foregroundColor(.white)
                }
        }
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            HStack(spacing: 8) {
                PinzButton(
                    type: .slot(style: .secondary(needBorder: true), title: PinzBaseStrings.AddMedia.Button.addMore),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .plain { isMorePickerPresented = true }
                )
                PinzButton(
                    type: .slot(style: .primary, title: PinzBaseStrings.Common.Button.next),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    action: .async { try await viewModel.asyncDispatch(.processGrouping) }
                )
                .disabledWithOpacity(viewModel.uploadedMediaEntries.isEmpty)
            }
        }
    }
}
