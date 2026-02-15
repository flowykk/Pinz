import Foundation

public final class SelectedTripStorage {
    public static let shared = SelectedTripStorage()
    
    private let userDefaults = UserDefaults.standard
    private let selectedTripKey = "selectedTripID"
    
    private init() {}
    
    public var selectedTripID: String? {
        get {
            userDefaults.string(forKey: selectedTripKey)
        }
        set {
            if let newValue {
                userDefaults.set(newValue, forKey: selectedTripKey)
            } else {
                userDefaults.removeObject(forKey: selectedTripKey)
            }
        }
    }
    
    public func selectTrip(id: String) {
        selectedTripID = id
    }
    
    public func clearSelection() {
        selectedTripID = nil
    }
}
