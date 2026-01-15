import SwiftUI
import PinzUI
import MapKit
import PinzNetworking
import PinzNavigation

@Observable
public class AppearanceViewModel {

    public enum PinzMapStyle: SegmentedItem {
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
    }

    public var state = State()
    
    private let networkService = NetworkService()

    public init() {}

    public func dispatch(_ intent: Intent) {
        switch intent {
        case let .changeMapStyle(mapStyle):
            withAnimation(.easeInOut(duration: 0.3)) {
                self.state.mapStyle = mapStyle
            }
        }
    }
}
