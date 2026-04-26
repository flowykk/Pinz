import SwiftUI
import PinzBase
import PinzDomain

public enum PrivacyIcon: String, Setting.Icon, SegmentedItem {
    case lockOpened = "lock.open.fill"
    case lockClosed = "lock.fill"

    public var id: Self { self }

    public var content: SegmentedItemContent {
        switch self {
        case .lockOpened: .icon(rawValue, PinzUIAsset.accentGreen.swiftUIColor)
        case .lockClosed: .icon(rawValue, PinzUIAsset.accentRed.swiftUIColor)
        }
    }

    public var apiValue: String { self == .lockOpened ? "Public" : "Private" }

    public static func from(isPrivate: Bool) -> PrivacyIcon { isPrivate ? .lockClosed : .lockOpened }
}

public struct PrivacySection: View {

    let onSelectionChanged: ((PrivacyIcon) -> Void)?
    @State private var privacySelection: PrivacyIcon

    public init(
        initialSelection: PrivacyIcon = .lockClosed,
        onSelectionChanged: ((PrivacyIcon) -> Void)? = nil
    ) {
        self._privacySelection = State(initialValue: initialSelection)
        self.onSelectionChanged = onSelectionChanged
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            SettingTitle(PinzBaseStrings.Trips.Header.membersAndPrivacy)
                .padding(.leading, 12)

            SegmentedPicker(selection: $privacySelection, items: [.lockOpened, .lockClosed])
                .onChange(of: privacySelection) { _, new in onSelectionChanged?(new) }
        }
    }
}
