import SwiftUI
import PinzDomain

public struct DatePickerView: View {

    @Binding var isPresented: Bool
    @Binding var date: Date?
    var pickerHeight: Binding<CGFloat>

    public init(
        isPresented: Binding<Bool>,
        date: Binding<Date?>,
        pickerHeight: Binding<CGFloat>
    ) {
        self._isPresented = isPresented
        self._date = date
        self.pickerHeight = pickerHeight
    }

    public var body: some View {
        VStack(alignment: .center) {
            Spacer()

            DatePicker(
                "",
                selection: Binding(
                    get: { date ?? .now },
                    set: { newDate in date = newDate }
                ),
                displayedComponents: [.date]
            )
            .datePickerStyle(.wheel)
            .labelsHidden()
            .measureHeight(for: pickerHeight)

            PinzButton(type: .slot(style: .primary, title: "Готово")) {
                isPresented = false
            }
            .padding(.horizontal, 12)
            .padding(.bottom, 8)
        }
    }
}

extension View {
    public func datePickerSheet(
        isPresented: Binding<Bool>,
        date: Binding<Date?>,
        pickerHeight: Binding<CGFloat>
    ) -> some View {
        self.sheet(isPresented: isPresented) {
            DatePickerView(
                isPresented: isPresented,
                date: date,
                pickerHeight: pickerHeight
            )
            .pinzSheet()
            .presentationDetents([.height(pickerHeight.wrappedValue + 70)])
        }
    }
}
