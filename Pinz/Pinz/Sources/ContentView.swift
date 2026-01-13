import SwiftUI
import PinzAuthentication

public struct ContentView: View {
    public init() {}

    public var body: some View {
//        AuthFlowView()
        SettingsView()
    }
}

struct ContentView_Previews: PreviewProvider {
    static var previews: some View {
        ContentView()
    }
}
