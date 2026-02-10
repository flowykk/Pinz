import SwiftUI
import MapKit

public extension Array where Element == Pin {
    func calculateInitialMapPosition() -> MapCameraPosition {
        guard !isEmpty else {
            return .automatic
        }
        
        let coordinates = map { $0.coordinates }
        
        let minLat = coordinates.map { $0.latitude }.min() ?? 0
        let maxLat = coordinates.map { $0.latitude }.max() ?? 0
        let minLon = coordinates.map { $0.longitude }.min() ?? 0
        let maxLon = coordinates.map { $0.longitude }.max() ?? 0
        
        let spanLat = (maxLat - minLat) * 1.5
        let spanLon = (maxLon - minLon) * 1.5

        let centerLat = (minLat + maxLat) / 2 + (spanLat * 0.1)
        let centerLon = (minLon + maxLon) / 2
        
        return .region(
            MKCoordinateRegion(
                center: CLLocationCoordinate2D(latitude: centerLat, longitude: centerLon),
                span: MKCoordinateSpan(
                    latitudeDelta: Swift.max(spanLat, 0.01),
                    longitudeDelta: Swift.max(spanLon, 0.01)
                )
            )
        )
    }
}
