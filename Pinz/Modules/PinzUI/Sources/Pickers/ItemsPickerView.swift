import SwiftUI
import PinzDomain

public struct ItemsPickerView<Item: PickerItem>: View {
    let items: [Item]
    @Binding var selection: Item
    let customizableItem: Item?
    let saveCustomizableItem: ((String) -> Void)?
    @Binding var isPresented: Bool

    @State var textFieldVisible: Bool = false
    @State var textFieldText: String = ""

    public init(
        items: [Item],
        selection: Binding<Item>,
        customizableItem: Item? = nil,
        saveCustomizableItem: ((String) -> Void)? = nil,
        isPresented: Binding<Bool>,
    ) {
        self.items = items
        self._selection = selection
        self.customizableItem = customizableItem
        self._isPresented = isPresented
        self.saveCustomizableItem = saveCustomizableItem
    }

    public var body: some View {
        VStack(alignment: .center) {
            Spacer()

            Picker("", selection: $selection) {
                ForEach(items) { item in
                    Text(item.value)
                }
            }
            .pickerStyle(.wheel)
            .labelsHidden()

            if textFieldVisible {
                SettingsGroup(
                    settings: [
                        .textField(Setting.TextFieldSetting(
                            id: "customTextField",
                            text: $textFieldText,
                            placeholder: "Введите нужное значение"
                        ))
                    ]
                ).padding(.horizontal, 12)
            }

            PinzButton(
                type: .slot(style: .primary, title: "Готово"),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor
            ) {
                isPresented = false
                if !textFieldText.isEmpty {
                    saveCustomizableItem?(textFieldText)
                }
            }
            .padding(.horizontal, 12)
            .padding(.bottom, 8)
        }
        .onAppear {
            if selection.isCustomizable {
                withAnimation(.easeInOut(duration: 0.3)) {
                    textFieldVisible = true
                    textFieldText = selection.value
                }
            }
        }
        .ifLet(customizableItem) { view, customizableItem in
            view.onChange(of: selection) { _, newValue in
                withAnimation(.easeInOut(duration: 0.3)) {
                    textFieldVisible = newValue.isCustomizable ? true : false
                }
            }
        }
    }
}

extension View {
    public func itemsPickerSheet<Item: PickerItem>(
        isPresented: Binding<Bool>,
        items: [Item],
        selection: Binding<Item>,
        pickerHeight: Binding<CGFloat>? = nil,
        customizableItem: Item? = nil,
        saveCustomizableItem: ((String) -> Void)? = nil
    ) -> some View {
        self.sheet(isPresented: isPresented) {
            ItemsPickerView(
                items: items,
                selection: selection,
                customizableItem: customizableItem,
                saveCustomizableItem: saveCustomizableItem,
                isPresented: isPresented,
            )
            .pinzSheet()
            .presentationDetents([.height(300)])
        }
    }
}
