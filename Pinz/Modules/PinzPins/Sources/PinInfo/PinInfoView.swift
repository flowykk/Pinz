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

    @Environment(\.appRouter) private var router

    var datesSettingValue: String {
        if let startDate = viewModel.pin.startDate, let endDate = viewModel.pin.endDate {
            "\(startDate.formattedToDayMonthYear) — \(endDate.formattedToDayMonthYear)"
        } else {
            "Не выбрано"
        }
    }

    public init(pin: Pin) {
        viewModel = PinInfoViewModel(pin: pin)
    }

    public var body: some View {
        VStack(spacing: 0) {
            header

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
                    .padding(.horizontal, 4)
            }
        }
        .onAppear { viewModel.setRouter(router) }
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
    }

    @ViewBuilder
    private var header: some View {
        switch viewModel.state {
        case .info, .gallery:
            Header(leftView: {
                PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.back)
                }
            }, centerView: {
                HeaderTitle(viewModel.pin.name, subtitle: viewModel.pin.category.value)
            }, rightView: {
                PinzButton(type: .icon(.warning), tint: PinzUIAsset.accentOrange.swiftUIColor) {

                }
                PinzButton(type: .icon(.pencil), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                    viewModel.dispatch(.edit)
                }
            }, additionalView: {
                SegmentedPicker(selection: $viewModel.state, items: [.info, .gallery])
            }, height: nil)
        case .editing:
            Header {
                PinzButton(type: .text("Отмена")) {
                    viewModel.dispatch(.endEdit)
                }
            } rightView: {
                PinzButton(type: .text("Готово")) {
                    viewModel.dispatch(.endEdit)
                }
            }
        }
    }

    var gallery: some View {
        ScrollView {
            PinzGrid($viewModel.pin.medias, columns: galleryColumns, spacing: 4) { media, index in
                switch media.content {
                case let .image(image):
                    mediaThumbnail(for: image)
                        .contextMenu {
                            Button {

                            } label: {
                                Label("Детали", systemImage: "eye.fill")
                            }

                            Divider()
                            Button(role: .destructive) {

                            } label: {
                                Label("Удалить", systemImage: "trash")
                            }
                        } preview: {
                            mediaThumbnail(for: image)
                        }
                case .video:
                    EmptyView()
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

    private func mediaThumbnail(for image: UIImage) -> some View {
        MediaThumbnailView(
            image: image,
            contentMode: .fit,
            cornerRadius: 12
        )
    }
}
