import SwiftUI
import PinzUI
import PinzDomain

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

    let gallerySpacing: CGFloat = 4

    @Environment(\.appRouter) private var router

    var datesSettingValue: String {
        if let startDate = viewModel.pin.startDate, let endDate = viewModel.pin.endDate {
            "\(startDate.formattedToDayMonthYear) — \(endDate.formattedToDayMonthYear)"
        } else {
            "Не выбрано"
        }
    }

    public init(pin: Pin, updateAction: PinUpdateAction? = nil) {
        viewModel = PinInfoViewModel(pin: pin, updateAction: updateAction)
    }

    public var body: some View {
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
        .onAppear { viewModel.setRouter(router) }
        .onDisappear { viewModel.onDisappear() }
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
        .fullScreenCover(isPresented: $isStoriesPresented) {
            PinStoryView(pins: [viewModel.pin])
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
            })
        case .editing:
            Header {
                PinzButton(
                    type: .text("Отмена"),
                    action: .plain { viewModel.dispatch(.endEdit) }
                )
            } rightView: {
                PinzButton(
                    type: .text("Готово"),
                    action: .plain { viewModel.dispatch(.endEdit) }
                )
            }
        }
    }

    var gallery: some View {
        ScrollView {
            PinzGrid($viewModel.pin.medias, columns: galleryColumns, spacing: gallerySpacing) { media, index in
                MediaItemThumbnailView(
                    mediaItem: media,
                    contentMode: .fit,
                    cornerRadius: 14
                ).onTapGesture {
                    viewModel.dispatch(.navigate(.mediaInfo(media)))
                }
            }
        }
        .scrollIndicators(.hidden)
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
}
