import SwiftUI
import PinzUI
import MapKit
import PinzNetworking

@Observable
public class AppearanceViewModel {

    public enum PinzMapStyle: String, SegmentedItem {
        case scheme
        case hybrid
        case satelight

        public var id: Self { self }

        public var title: String {
            switch self {
            case .scheme: "Схема"
            case .satelight: "Спутник"
            case .hybrid: "Гибрид"
            }
        }

        public func toMapKitMapStyle() -> MapStyle {
            switch self {
            case .scheme: .standard
            case .satelight: .imagery
            case .hybrid: .hybrid
            }
        }
    }

    public struct State {
        var mapStyle: PinzMapStyle = .satelight
    }

    public enum Intent {
        case changeMapStyle(PinzMapStyle)
        case loadMapStyle
        case saveMapStyle
    }

    public var state = State()
    
    private let networkService = NetworkService()
    private let userDefaults = UserDefaults.standard
    private let mapStyleKey = "pinzMapStyle"

    public init() {
        dispatch(.loadMapStyle)
    }

    public func dispatch(_ intent: Intent) {
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
