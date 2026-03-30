import SwiftUI
import PinzDomain

struct MergePinsPickerView: View {

    let pins: [RawPin]
    @Binding var isPresented: Bool
    let onMerge: (Int, Int) -> Void

    @State private var firstIndex: Int = 0
    @State private var secondIndex: Int = 1

    var body: some View {
        VStack(spacing: 0) {
            Text("Выбери пины, которые хочешь объединить")
                .roundedFount(size: 16)
                .padding(.top, 12)
                .padding(.bottom, 8)

            HStack(spacing: 0) {
                VStack(spacing: 0) {
                    Picker("", selection: $firstIndex) {
                        ForEach(pins.indices, id: \.self) { i in
                            Text("Пин \(i + 1)").tag(i)
                        }
                    }
                    .pickerStyle(.wheel)
                    .labelsHidden()
                }

                VStack(spacing: 0) {
                    Picker("", selection: $secondIndex) {
                        ForEach(pins.indices, id: \.self) { i in
                            Text("Пин \(i + 1)").tag(i)
                        }
                    }
                    .pickerStyle(.wheel)
                    .labelsHidden()
                }
            }

            PinzButton(
                type: .slot(style: .primary, title: "Объединить пины \(firstIndex + 1) и \(secondIndex + 1)"),
                disabled: firstIndex == secondIndex,
                action: .plain {
                    onMerge(firstIndex, secondIndex)
                    isPresented = false
                }
            )
            .padding(.horizontal, 12)
            .padding(.bottom, 8)
        }.animation(.default)
    }
}

extension View {
    public func mergePinsSheet(
        isPresented: Binding<Bool>,
        pins: [RawPin],
        onMerge: @escaping (Int, Int) -> Void
    ) -> some View {
        self.sheet(isPresented: isPresented) {
            MergePinsPickerView(
                pins: pins,
                isPresented: isPresented,
                onMerge: onMerge
            )
            .pinzSheet()
            .presentationDetents([.height(240)])
        }
    }

}
