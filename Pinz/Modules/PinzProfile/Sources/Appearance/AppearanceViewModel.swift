import SwiftUI
import PinzUI
import MapKit
import PinzNetworking

@Observable
final class AppearanceViewModel {

    enum PinzMapStyle: String, SegmentedItem {
        case scheme
        case hybrid
        case satelight

        var id: Self { self }

        var content: SegmentedItemContent {
            switch self {
            case .scheme: .text("Схема")
            case .satelight: .text("Спутник")
            case .hybrid: .text("Гибрид")
            }
        }

        func toMapKitMapStyle() -> MapStyle {
            switch self {
            case .scheme: .standard
            case .satelight: .imagery
            case .hybrid: .hybrid
            }
        }
    }

    struct State {
        var mapStyle: PinzMapStyle = .satelight
    }

    enum Intent {
        case changeMapStyle(PinzMapStyle)
        case loadMapStyle
        case saveMapStyle
    }

    var state = State()
    
    private let networkService = NetworkService()
    private let userDefaults = UserDefaults.standard
    private let mapStyleKey = "pinzMapStyle"

    init() {
        dispatch(.loadMapStyle)
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case let .changeMapStyle(mapStyle):
            withAnimation(.easeInOut(duration: 0.3)) {
                state.mapStyle = mapStyle
            }
            dispatch(.saveMapStyle)
            
        case .loadMapStyle:
            if let savedStyle = userDefaults.string(forKey: mapStyleKey),
               let mapStyle = PinzMapStyle(rawValue: savedStyle) {
                state.mapStyle = mapStyle
            }
            
        case .saveMapStyle:
            userDefaults.set(state.mapStyle.rawValue, forKey: mapStyleKey)
        }
    }
}
