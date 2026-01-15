import SwiftUI

public protocol SegmentedItem: Identifiable, Equatable {
    var title: String { get }
}

@MainActor
public struct SegmentedPicker<Item: SegmentedItem>: View {
    @Binding var selection: Item
    var items: [Item]

    public init(selection: Binding<Item>, items: [Item]) {
        self._selection = selection
        self.items = items
    }

    public var body: some View {
        ZStack(alignment: .topLeading) {
            HStack(spacing: 4) {
                ForEach(items.reversed().drop(while: { $0.id != selection.id }).dropFirst()) {
                    makeItemView($0)
                }.hidden()

                makeItemView(selection).hidden().background {
                    RoundedRectangle(cornerRadius: 10)
                        .foregroundColor(PinzUIAsset.background.swiftUIColor)
                }

                ForEach(items.drop(while: { $0.id != selection.id }).dropFirst()) {
                    makeItemView($0)
                }.hidden()
            }

            HStack(spacing: 4) {
                ForEach(items) { item in
                    makeItemView(item).simultaneousGesture(TapGesture().onEnded {
                        if selection.id != item.id {
                            withAnimation(.spring(response: 0.4)) {
                                selection = item
                            }
                        }})
                }
            }
        }
        .padding(3)
        .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
        .cornerRadius(12)
    }

    private func makeItemView(_ item: Item) -> some View {
        HStack(spacing: 0) {
            Spacer(minLength: 0)
            Text(item.title)
                .roundedFount(size: 16)
            Spacer(minLength: 0)
        }
        .padding(.vertical, 8)
        .contentShape(Rectangle())
    }
}
