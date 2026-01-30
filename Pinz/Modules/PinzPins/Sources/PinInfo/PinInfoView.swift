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
                    viewModel.dispatch(.changeState(.editing))
                }
            }, additionalView: {
                SegmentedPicker(selection: $viewModel.state, items: [.info, .gallery])
            }, height: nil)
        case .editing:
            Header {
                PinzButton(type: .text("Отмена")) {
                    viewModel.dispatch(.changeState(.info))
                }
            } rightView: {
                PinzButton(type: .text("Готово")) {
                    viewModel.dispatch(.changeState(.info))
                }
            }
        }
    }

    var gallery: some View {
        return ScrollView {
            PinterestLikeGrid($viewModel.pin.medias, columns: 3, spacing: 4) { media, index in
                switch media.content {
                case let .image(image):
                    Image(uiImage: image)
                        .resizable()
                        .scaledToFit()
                        .cornerRadius(12)
                case .video:
                    EmptyView()
                }
            }
        }
        .scrollIndicators(.hidden)
    }
}

/**
 Custom SwiftUI view that displays data in a Pinterest-like grid style.

 - Important: Items in the data array must conform to the `Hashable` protocol.

 - Parameters:
 - data: Binding array of items to be displayed on the list. The item must conform to the `Hashable` protocol.
 - columns: The number of columns in the grid. The default value is 2.
 - content: View that represents each item in the grid. It provides the current item of the data array and its index.

 - Example:

 ````
 struct ContentView: View {
 @State var data = [1,2,3,4]

 var body: some View {
 PinterestLikeGrid($data) { item, index in
 Text("\(item)")
 }
 }
 }
 ````
 */

@available(macOS 10.15, *)
public struct PinterestLikeGrid<T:Hashable, Content:View>: View {

    /// Binding to an array of hashable items to be displayed
    @Binding var data: [T]

    /// The number of columns in the grid
    let columns: Int

    /// Closure that takes as input a hashable item from the data array and its optional index, and returns a SwiftUI view that represents the item in the grid.
    let content: (_ item: T, _ index: Int?) -> Content

    /// A range representing the indices of the grid columns.
    let range: ClosedRange<Int>

    /// An array of arrays of hashable items that represents the data array splitted into the number of columns.
    var splittedData: [[T]] {
        Helper.splitData(from: data, into: columns)
    }

    var rowSpacing: CGFloat

    var columnSpacing: CGFloat

    /**
     Creates a new PinteresLikeGrid with the specified data, number of columns, row and column spacing and content. If column is nil, 2 is the default value.
     */
    public init(_ data: Binding<[T]>, columns: Int = 2, rowSpacing: CGFloat = 8, columnSpacing: CGFloat = 8, @ViewBuilder content: @escaping (_ item: T,  _ index: Int?) -> Content) {
        self._data = data
        self.columns = columns
        self.rowSpacing = rowSpacing
        self.columnSpacing = columnSpacing
        self.range = 0...(columns + 1)
        self.content = content
    }

    /**
     Creates a new PinteresLikeGrid with the specified data, number of columns, vertical and horizontal spacing and content. If column is nil, 2 is the default value.
     */
    public init(_ data: Binding<[T]>, columns: Int = 2, spacing: CGFloat = 8, @ViewBuilder content: @escaping (_ item: T,  _ index: Int?) -> Content) {
        self._data = data
        self.columns = columns
        self.rowSpacing = spacing
        self.columnSpacing = spacing
        self.range = 0...(columns + 1)
        self.content = content
    }

    public var body: some View {
        HStack(alignment: .top, spacing: columnSpacing) {
            ForEach(range, id: \.self) { index in
                if index < splittedData.count {
                    VStack(spacing: rowSpacing) {
                        ForEach(splittedData[index], id: \.self) { item in
                            content(item, getIndexInList(for: item))
                                .transition(.identity)
                        }
                    }
                }
            }
        }
        .animation(.easeInOut, value: data)
    }

    /**
     Returns the index of a specified item from the data array.
     - Parameter item: The item to search for the index.
     - Returns: The index of the specified item in the data array or nil, if it's not found.
     */
    private func getIndexInList(for item: T) -> Int? {
        data.firstIndex(where: { $0.hashValue == item.hashValue })
    }
}

class Helper {
    /**
     Splits the given array into multiple arrays with a specified number of columns.

     - Parameters:
     - arr: Array of any type
     - columnNumber: Number of columns

     - Returns: An array splitted into a given number of columns.
     */

    static func splitData<T>(from arr: [T], into columnNumber: Int = 2) -> [[T]] {
        guard !arr.isEmpty else { return [] }
        let columns = columnNumber > arr.count ? arr.count : columnNumber
        var result = [[T]](repeating: [], count: columns)

        for (index, item) in arr.enumerated() {
            let arrayIndex = index % columns
            result[arrayIndex].append(item)
        }
        return result
    }
}
