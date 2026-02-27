import SwiftUI
import PinzUI

struct UnselectedTripView: View {
    var body: some View {
        VStack() {
            Group {
                Text("Ты ещё не выбрал путешествие")
                Text("Нажми на \(Image(systemName: "chevron.down")), чтобы выбрать его из существуюших или создать новое")
            }.multilineTextAlignment(.center)
        }
        .roundedFount(size: 18)
        .padding(.horizontal, 12)
    }
}
