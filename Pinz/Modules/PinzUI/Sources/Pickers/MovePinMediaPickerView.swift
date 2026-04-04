import SwiftUI
import PinzBase
import PinzDomain

struct MovePinMediaPickerView: View {

    let movablePins: [(globalIndex: Int, pin: RawPin)]
    @Binding var isPresented: Bool
    let onMove: (Int) -> Void

    @State private var selectedIndex: Int = 0

    var body: some View {
        VStack(spacing: 0) {
            Text(PinzBaseStrings.MoveMedia.Label.instructions)
                .multilineTextAlignment(.center)
                .roundedFount(size: 16)
                .padding(.top, 12)
                .padding(.bottom, 8)

            Picker("", selection: $selectedIndex) {
                ForEach(movablePins.indices, id: \.self) { i in
                    Text(PinzBaseStrings.Common.Label.pinNumber(movablePins[i].globalIndex + 1)).tag(i)
                }
            }
            .pickerStyle(.wheel)
            .labelsHidden()
            .animation(.easeInOut(duration: 0.25), value: selectedIndex)

            PinzButton(
                type: .slot(style: .primary, title: PinzBaseStrings.MoveMedia.Button.moveToPinNumber(movablePins[safe: selectedIndex].map { $0.globalIndex + 1 } ?? 0)),
                disabled: movablePins.isEmpty,
                action: .plain {
                    if let target = movablePins[safe: selectedIndex] {
                        onMove(target.globalIndex)
                    }
                    isPresented = false
                }
            )
            .padding(.horizontal, 12)
            .padding(.bottom, 8)
        }
    }
}

extension View {
    public func movePinMediaSheet(
        isPresented: Binding<Bool>,
        movablePins: [(globalIndex: Int, pin: RawPin)],
        onMove: @escaping (Int) -> Void
    ) -> some View {
        self.sheet(isPresented: isPresented) {
            MovePinMediaPickerView(
                movablePins: movablePins,
                isPresented: isPresented,
                onMove: onMove
            )
            .pinzSheet()
            .presentationDetents([.height(220)])
        }
    }
}

private extension Array {
    subscript(safe index: Int) -> Element? {
        indices.contains(index) ? self[index] : nil
    }
}
