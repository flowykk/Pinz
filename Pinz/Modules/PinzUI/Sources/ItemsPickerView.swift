import SwiftUI
import PinzDomain

public struct ItemsPickerView<Item: PickerItem>: View {

    var items: [Item]
    @Binding var selection: Item
    @Binding var isPresented: Bool

    public init(
        items: [Item],
        selection: Binding<Item>,
        isPresented: Binding<Bool>
    ) {
        self.items = items
        self._selection = selection
        self._isPresented = isPresented
    }

    public var body: some View {
        VStack {
            Spacer()
            
            Picker("", selection: $selection) {
                ForEach(items) { item in
                    Text(item.value)
                }
            }.pickerStyle(.wheel)

            PinzButton(
                type: .slot("Готово", PinzUIAsset.textPrimary.swiftUIColor),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor
            ) {
                isPresented = false
            }.padding(.horizontal, 12)
        }
    }
}

extension View {
    public func itemsPickerSheet<Item: PickerItem>(
        isPresented: Binding<Bool>,
        items: [Item],
        selection: Binding<Item>
    ) -> some View {

        self.sheet(isPresented: isPresented) {
            ItemsPickerView(
                items: items,
                selection: selection,
                isPresented: isPresented
            )
            .pinzSheet()
            .presentationDetents([.height(200)])
        }
    }
}
