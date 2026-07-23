import SwiftUI
import PinzDomain

public struct PinStoryView: View {

    var pins: [Pin]
    @State var currentStory: String

    public init(pins: [Pin]) {
        self.pins = pins
        self.currentStory = pins.first?.id ?? ""
    }

    public var body: some View {
        TabView(selection: $currentStory) {
            ForEach(pins) { pin in
                PinStoryCardView(pin: pin, pins: pins, currentStory: $currentStory)
            }
        }
        .tabViewStyle(.page(indexDisplayMode: .never))
        .background(.black)
        .ignoresSafeArea(edges: .bottom)
    }
}
