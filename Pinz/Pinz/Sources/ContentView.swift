import SwiftUI
import PinzAuthentication
import PinzProfile

public struct ContentView: View {
    public init() {}

    public var body: some View {
//        AuthFlowView()
//        SettingsView()
        ProfileView()
    }
}

struct ContentView_Previews: PreviewProvider {
    static var previews: some View {
        ContentView()
    }
}
