import SwiftUI
import Authentication

public struct ContentView: View {
    public init() {}

    public var body: some View {
        AuthFlowView()
    }
}

struct ContentView_Previews: PreviewProvider {
    static var previews: some View {
        ContentView()
    }
}
