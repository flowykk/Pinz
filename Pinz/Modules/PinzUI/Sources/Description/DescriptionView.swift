import SwiftUI
import PinzBase

enum DescriptionIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"
    case text = "text.alignleft"
}

public struct DescriptionView: View {
    
    private let title: String
    private let description: String?
    private let collapseButtonText: (collapsed: String, expanded: String)
    private let onTapAction: (() -> Void)?

    @State private var isCollapsed: Bool = true
    
    public init(
        title: String = PinzBaseStrings.Common.Label.description,
        description: String?,
        collapseButtonText: (collapsed: String, expanded: String) = (
            PinzBaseStrings.Common.Button.expand,
            PinzBaseStrings.Common.Button.collapse
        ),
        onAddAction: (() -> Void)? = nil
    ) {
        self.title = title
        self.description = description
        self.collapseButtonText = collapseButtonText
        self.onTapAction = onAddAction
    }
    
    public var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 0) {
                SettingTitle(title)
                if let description, !description.isEmpty {
                    Spacer()
                    Button {
                        withAnimation(.easeInOut(duration: 0.3)) {
                            isCollapsed.toggle()
                        }
                    } label: {
                        HStack(spacing: 4) {
                            Text(isCollapsed ? collapseButtonText.collapsed : collapseButtonText.expanded)
                            Image(systemName: "chevron.down")
                                .rotationEffect(.degrees(isCollapsed ? 0 : 180))
                        }
                        .roundedFount(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                    }
                }
            }
            .padding(.bottom, 6)
            .padding(.leading, 12)
            .padding(.trailing, 16)
            
            if let description, !description.isEmpty {
                VStack(spacing: 0) {
                    Text(description)
                        .roundedFount(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                        .lineLimit(isCollapsed ? 5 : nil)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(.horizontal, 16)
                        .padding(.vertical, 16)
                }
                .frame(minHeight: 52)
                .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
                .cornerRadius(26)
            } else {
                SettingsGroup(settings: [
                    .default(Setting.DefaultSetting(
                        id: "tripDescription",
                        leading: .iconTitle(DescriptionIcon.text, PinzBaseStrings.Common.Button.addDescription),
                        trailing: .icon(DescriptionIcon.chevronRight),
                        action: onTapAction.flatMap {
                            action in .plain { action() }
                        } ?? nil
                    ))
               ])
            }
        }
    }
}
