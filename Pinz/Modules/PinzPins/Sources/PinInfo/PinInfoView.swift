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

            ScrollView {
                VStack(spacing: 0) {
                    switch viewModel.state {
                    case .info, .editing:
                        settings
                            .padding(.horizontal, 12)
                    case .gallery:
                        gallery
                            .padding(.horizontal, 4)
                    }
                    map.if(viewModel.state != .info) { view in view.hidden() }
                        .padding(.top, 12)
                }

                Spacer()
            }
            .scrollIndicators(.hidden)
            .scrollDisabled(viewModel.isEditing)
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
        return ScrollView {
            PinzGrid($viewModel.pin.medias, columns: 3, spacing: 4) { media, index in
                switch media.content {
                case let .image(image):
                    Image(uiImage: image)
                        .resizable()
                        .scaledToFit()
                        .cornerRadius(12)
                        .contextMenu {
                            Button {

                            } label: {
                                Label("Переместить", systemImage: "arrow.left.arrow.right")
                            }

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
                            Image(uiImage: image)
                                .resizable()
                                .scaledToFit()
                                .cornerRadius(12)
                        }
                case .video:
                    EmptyView()
                }
            }
        }
        .scrollIndicators(.hidden)
    }
}
