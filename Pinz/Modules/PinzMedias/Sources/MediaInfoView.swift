import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

enum MediaInfoIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"
    case trash = "trash"
}

public struct MediaInfoView: View {
    
    private let media: MediaItem
    
    @Environment(\.appRouter) private var router
    
    public init(media: MediaItem) {
        self.media = media
    }
    
    public var body: some View {
        VStack(spacing: 0) {
            header
            
            ScrollView {
                VStack(spacing: 12) {
                    switch media.type {
                    case .image:
                        LoadableImageThumbnail(url: media.mediaURL) { state in mediaView(for: state) }
                    case .video:
                        LoadableVideoThumbnail(url: media.mediaURL) { state in mediaView(for: state) }
                    }

                    privacy

                    delete
                }
            }
            .scrollIndicators(.hidden)
            .padding(.top, 8)
            .padding(.horizontal, 12)
        }
        .background(PinzUIAsset.background.swiftUIColor)
    }

    @ViewBuilder
    private func mediaView(for state: LoadableMediaState) -> some View {
        switch state {
        case .empty:
            Rectangle()
                .fill(Color.gray.opacity(0.3))
                .aspectRatio(1, contentMode: .fit)
                .overlay {
                    ProgressView()
                        .tint(.white)
                }
                .cornerRadius(16)
        case .ready(let image):
            Image(uiImage: image)
                .resizable()
                .aspectRatio(contentMode: .fit)
                .cornerRadius(16)
        case .failure:
            Rectangle()
                .fill(Color.gray.opacity(0.3))
                .aspectRatio(1, contentMode: .fit)
                .overlay {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundColor(.white)
                }
                .cornerRadius(16)
        }
    }
    
    private var header: some View {
        Header(leftView: {
            PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                router?.pop()
            }
        }, rightView: {
            PinzButton(type: .icon(.crop), tint: PinzUIAsset.textPrimary.swiftUIColor) {

            }
            PinzButton(type: .icon(.download), tint: PinzUIAsset.textPrimary.swiftUIColor) {

            }
        })
    }

    private var privacy: some View {
        PrivacySection(members: TripMember.stubs())
    }

    private var delete: some View {
        SettingsGroup(settings: [
            .default(Setting.DefaultSetting(
                id: "mediaDelete",
                leading: .iconTitle(MediaInfoIcon.trash, "Удалить медиа"),
                trailing: .icon(MediaInfoIcon.chevronRight),
                style: .destructive,
                action: .plain { }
            ))
        ])
    }
}
