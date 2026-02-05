import SwiftUI
import PinzUI
import PinzNetworking
import PinzBase

@Observable
final class AppearanceViewModel {

    struct State {
        var mapStyle: PinzMapStyle = .satelight
    }

    enum Route {
        case back
    }

    enum Intent {
        case navigate(Route)
        case changeMapStyle(PinzMapStyle)
        case loadMapStyle
        case saveMapStyle
    }

    var state = State()
    
    private let networkService = NetworkService()
    private let userDefaults = UserDefaults.standard
    private var router: AppRouting?

    init() {
        dispatch(.loadMapStyle)
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .navigate(route):
            switch route {
            case .back:
                router?.pop()
            }
        case let .changeMapStyle(mapStyle):
            withAnimation(.easeInOut(duration: 0.3)) {
                state.mapStyle = mapStyle
            }
            dispatch(.saveMapStyle)
            
        case .loadMapStyle:
            if let savedStyle = userDefaults.string(forKey: PinzMapStyle.mapStyleKey),
               let mapStyle = PinzMapStyle(rawValue: savedStyle) {
                state.mapStyle = mapStyle
            }
            
        case .saveMapStyle:
            userDefaults.set(state.mapStyle.rawValue, forKey: PinzMapStyle.mapStyleKey)
        }
    }

    public func setRouter(_ router: AppRouting?) {
        self.router = router
    }
}
