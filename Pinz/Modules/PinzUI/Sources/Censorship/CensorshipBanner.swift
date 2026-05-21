import SwiftUI
import PinzBase

public struct CensorshipBanner: View {

    public enum Field {
        case name
        case description

        var text: String {
            switch self {
            case .name: PinzBaseStrings.Censorship.Banner.name
            case .description: PinzBaseStrings.Censorship.Banner.description
            }
        }
    }

    private let field: Field

    public init(field: Field) {
        self.field = field
    }

    public var body: some View {
        HStack(alignment: .top, spacing: 8) {
            Image(systemName: "exclamationmark.triangle.fill")
                .foregroundStyle(PinzUIAsset.textSecondary.swiftUIColor)
                .font(.system(size: 14))
                .padding(.top, 1)
            Text(field.text)
                .roundedFont(
                    size: 13,
                    weight: .regular,
                    foregroundColor: PinzUIAsset.textSecondary.swiftUIColor
                )
                .fixedSize(horizontal: false, vertical: true)
            Spacer(minLength: 0)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
        .clipShape(RoundedRectangle(cornerRadius: 10))
    }
}
