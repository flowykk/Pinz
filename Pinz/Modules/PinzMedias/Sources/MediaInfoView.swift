import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

enum MediaInfoIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"
    case trash = "trash"
}

public struct MediaInfoView: View {
    
    private let media: LoadedMedia
    
    @Environment(\.appRouter) private var router
    
    public init(media: LoadedMedia) {
        self.media = media
    }
    
    public var body: some View {
        VStack(spacing: 0) {
            header
            
            ScrollView {
                VStack(spacing: 12) {
                    switch media.content {
                    case let .image(image):
                        Image(uiImage: image)
                            .resizable()
                            .aspectRatio(contentMode: .fit)
                            .cornerRadius(16)
                    case let .video(_, firstFrame):
                        Image(uiImage: firstFrame)
                            .resizable()
                            .aspectRatio(contentMode: .fit)
                            .cornerRadius(16)
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
        PrivacySection(
            members: [
                TripMember(
                    isPrivate: true,
                    username: "danuwka",
                    avatar: PinzUIAsset.media3.image
                ),
                TripMember(
                    isPrivate: false,
                    username: "kostik",
                    avatar: PinzUIAsset.media10.image
                ),
                TripMember(
                    isPrivate: false,
                    username: "dimka",
                    avatar: PinzUIAsset.media5.image
                ),
            ]
        )
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
