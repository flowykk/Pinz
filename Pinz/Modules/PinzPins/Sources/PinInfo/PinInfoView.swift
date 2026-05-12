import SwiftUI
import PinzUI
import PinzDomain
import PinzBase
import PinzAccessibility

enum PinInfoIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"

    case warning = "exclamationmark.triangle.fill"
    case checkmark = "checkmark.circle.fill"

    case info = "info.circle.fill"
    case calendar = "calendar"

    case trash = "trash"
}

public struct PinInfoView: View {

    @State var viewModel: PinInfoViewModel
    
    @State var isDescriptionCollapsed = true
    @State var isCategoryPickerPresented = false
    @State var isStartDatePickerPresented = false
    @State var isEndDatePickerPresented = false
    @State private var datePickerHeight: CGFloat = 0
    
    @State private var galleryColumns = 3
    @State private var magnificationScale: CGFloat = 1.0
    @State private var isStoriesPresented = false
    @State var showDeletePinAlert = false

    let gallerySpacing: CGFloat = 4

    @Environment(\.appRouter) private var router
    @Environment(\.showToast) private var showToast

    var datesSettingValue: String {
        if let startDate = viewModel.pin.startDate, let endDate = viewModel.pin.endDate {
            return "\(startDate.formattedToDayMonthYear) — \(endDate.formattedToDayMonthYear)"
        } else {
            return PinzBaseStrings.Common.Label.notSelected
        }
    }

    public init(pin: Pin, updateAction: PinUpdateAction? = nil, deleteAction: PinDeleteAction? = nil) {
        viewModel = PinInfoViewModel(pin: pin, updateAction: updateAction, deleteAction: deleteAction)
    }

    public var body: some View {
        ZStack(alignment: .bottom) {
            CollapsibleHeader(needsBlur: viewModel.state == .gallery ? true : false) {
                header
            } stickyHeader: {
                if !viewModel.isEditing {
                    SegmentedPicker(selection: $viewModel.state, items: [.info, .gallery])
                        .padding(.horizontal, 12)
                }
            } content: {
                if viewModel.state == .info || viewModel.state == .editing {
                    ScrollView {
                        VStack(spacing: 0) {
                            settings
                                .padding(.horizontal, 12)
                            map.if(viewModel.state != .info) { view in view.hidden() }
                                .padding(.top, 12)
                                .padding(.horizontal, 12)
                        }
                    }
                    .scrollIndicators(.hidden)
                    .scrollDisabled(viewModel.isEditing)
                } else {
                    gallery
                        .padding(.horizontal, gallerySpacing)
                }
            }

            if viewModel.state == .gallery, !viewModel.isEditing {
                galleryAddMediaBar
            }
        }
        .onAppear {
            viewModel.setRouter(router)
            viewModel.setToast(showToast)
            viewModel.refreshPinUploadAdditionSessionFlag()
            router?.setPinUploadAdditionSuccessHandler { pin in
                viewModel.applyPinAfterAdditionUpload(pin)
            }
        }
        .onChange(of: viewModel.state) { _, newState in
            if newState == .gallery {
                viewModel.refreshPinUploadAdditionSessionFlag()
            }
        }
        .onDisappear {
            viewModel.onDisappear()
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .itemsPickerSheet(
            isPresented: $isCategoryPickerPresented,
            items: PinCategory.allCases,
            selection: $viewModel.pin.category,
            customizableItem: .custom(),
            saveCustomizableItem: { value in
                viewModel.pin.category = .custom(value)
            }
        )
        .datePickerSheet(
            isPresented: $isStartDatePickerPresented,
            date: $viewModel.pin.startDate,
            pickerHeight: $datePickerHeight
        )
        .datePickerSheet(
            isPresented: $isEndDatePickerPresented,
            date: $viewModel.pin.endDate,
            pickerHeight: $datePickerHeight
        )
        .onChange(of: isStartDatePickerPresented) { _, isPresented in
            if !isPresented, let error = viewModel.validateDates() { showToast(error) }
        }
        .onChange(of: isEndDatePickerPresented) { _, isPresented in
            if !isPresented, let error = viewModel.validateDates() { showToast(error) }
        }
        .fullScreenCover(isPresented: $isStoriesPresented) {
            PinStoryView(pins: [viewModel.pin])
        }
        .alert(PinzBaseStrings.PinInfo.Alert.DeletePin.title, isPresented: $showDeletePinAlert) {
            Button(PinzBaseStrings.PinInfo.Alert.DeletePin.confirm, role: .destructive) {
                viewModel.dispatch(.deletePin)
            }
            Button(PinzBaseStrings.Common.Button.cancel, role: .cancel) {}
        } message: {
            Text(PinzBaseStrings.PinInfo.Alert.DeletePin.message)
        }
    }

    @ViewBuilder
    private var header: some View {
        switch viewModel.state {
        case .info, .gallery:
            Header(leftView: {
                PinzButton(
                    type: .icon(.chevronLeft),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.navigate(.back)) }
                )
            }, centerView: {
                HeaderTitle(viewModel.pin.name, subtitle: viewModel.pin.category.value)
                    .pinzA11y(.pin(.row(.headerTitleDetail)))
            }, rightView: {
                PinzButton(
                    type: .icon(.stories),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { isStoriesPresented = true }
                )
//                PinzButton(type: .icon(.warning), tint: PinzUIAsset.accentOrange.swiftUIColor) {
//
//                }
                PinzButton(
                    type: .icon(.pencil),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { viewModel.dispatch(.edit) }
                )
                .pinzA11y(.pin(.button(.edit)))
            })
        case .editing:
            Header {
                PinzButton(
                    type: .text(PinzBaseStrings.Common.Button.cancel),
                    action: .plain { viewModel.dispatch(.cancelEdit) }
                )
                .pinzA11y(.pin(.button(.cancel)))
            } rightView: {
                PinzButton(
                    type: .text(PinzBaseStrings.Common.Button.done),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    disabled: viewModel.isSaving,
                    action: .async {
                        await viewModel.asyncDispatch(.saveEdits)
                    }
                )
                .pinzA11y(.pin(.button(.done)))
            }
        }
    }

    var gallery: some View {
        ScrollView {
            PinzGrid($viewModel.pin.medias, columns: galleryColumns, spacing: gallerySpacing) { media, _ in
                MediaItemThumbnailView(
                    mediaItem: media,
                    contentMode: .fit,
                    cornerRadius: 14,
                    onMediaUpdated: { viewModel.applyGalleryMediaPrivacyUpdate($0) },
                    showsPinMediaDeleteControl: viewModel.canDeletePinMediaFromGallery(media),
                    pinMediaDeleteBusy: viewModel.pendingDeleteMediaId == media.mediaId,
                    onPinMediaDelete: { viewModel.deletePinMediaFromGallery(media) },
                    pinIdForServerMediaDelete: viewModel.pin.serverId,
                    pinResponseAction: viewModel.pinResponseActionForCurrentPin()
                ).onTapGesture {
                    viewModel.dispatch(.navigate(.mediaInfo(media)))
                }
            }
        }
        .scrollIndicators(.hidden)
        .padding(.bottom, 90)
        .simultaneousGesture(
            MagnificationGesture()
                .onChanged { value in
                    magnificationScale = value

                    let targetColumns: Int
                    if value >= 2.0 {
                        targetColumns = 1
                    } else if value >= 1.6 {
                        targetColumns = 2
                    } else if value >= 1.2 {
                        targetColumns = 3
                    } else if value >= 0.9 {
                        targetColumns = 4
                    } else {
                        targetColumns = 5
                    }

                    if targetColumns != galleryColumns {
                        withAnimation(.easeOut(duration: 0.4)) {
                            galleryColumns = targetColumns
                        }
                    }
                }
                .onEnded { _ in
                    withAnimation(.easeOut(duration: 0.4)) {
                        magnificationScale = 1.0
                    }
                }
        )
    }

    private var galleryAddMediaBar: some View {
        BottomGradientWithButtons {
            VStack(spacing: 4) {
                PinzButton(
                    type: .slot(style: .primary, title: PinzBaseStrings.PinUpload.Header.addMedia),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    disabled: viewModel.addMediaButtonDisabled,
                    action: .async {
                        await viewModel.asyncDispatch(.startAddMedia)
                    }
                )
                .pinzA11y(.pin(.button(.addMedia)))
                if viewModel.hasActivePinUploadAdditionSession {
                    Text(PinzBaseStrings.PinUpload.Loading.uploading)
                        .roundedFont(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                        .multilineTextAlignment(.center)
                }
            }
        }
    }
}
