import SwiftUI

public enum PickerItemContent {
    case text(String)
    case icon(String, Color)
}

public protocol PickerItem: Identifiable, Hashable {
    var content: PickerItemContent { get }
    var value: String { get }
    var isCustomizable: Bool { get }
}
