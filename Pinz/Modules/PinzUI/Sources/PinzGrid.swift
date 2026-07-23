import SwiftUI

public struct PinzGrid<T: Hashable, Content: View>: View {

    @Binding var data: [T]

    let columns: Int
    let content: (_ item: T, _ index: Int?) -> Content
    let range: ClosedRange<Int>
    var rowSpacing: CGFloat
    var columnSpacing: CGFloat

    var splittedData: [[T]] {
        PinzGridHelper.splitData(from: data, into: columns)
    }

    public init(
        _ data: Binding<[T]>,
        columns: Int = 2,
        rowSpacing: CGFloat = 8,
        columnSpacing: CGFloat = 8,
        @ViewBuilder content: @escaping (_ item: T,  _ index: Int?) -> Content
    ) {
        self._data = data
        self.columns = columns
        self.rowSpacing = rowSpacing
        self.columnSpacing = columnSpacing
        self.range = 0...(columns + 1)
        self.content = content
    }

    public init(
        _ data: Binding<[T]>,
        columns: Int = 2,
        spacing: CGFloat = 8,
        @ViewBuilder content: @escaping (_ item: T,  _ index: Int?) -> Content
    ) {
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
                                .transition(
                                    .asymmetric(
                                        insertion: .opacity
                                            .combined(with: .scale(scale: 0.94, anchor: .center)),
                                        removal: .opacity
                                            .combined(with: .scale(scale: 0.86, anchor: .center))
                                    )
                                )
                        }
                    }
                }
            }
        }
        .animation(.spring(response: 0.44, dampingFraction: 0.82, blendDuration: 0.12), value: data)
    }

    private func getIndexInList(for item: T) -> Int? {
        data.firstIndex(where: { $0.hashValue == item.hashValue })
    }
}

class PinzGridHelper {
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
