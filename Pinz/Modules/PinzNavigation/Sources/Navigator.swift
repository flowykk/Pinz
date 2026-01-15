import Foundation
import SwiftUI

@Observable
public class Navigator<Destination: Hashable> {
    public var path: [Destination] = []
    
    public init() {}
    
    public func navigate(to destination: Destination) {
        path.append(destination)
    }
    
    public func back() {
        guard !path.isEmpty else { return }
        path.removeLast()
    }
    
    public func backToRoot() {
        path.removeAll()
    }
    
    public func replace(with destination: Destination) {
        path.removeLast()
        path.append(destination)
    }
}
