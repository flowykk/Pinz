import SwiftUI
import PinzUI
import PinzDomain
import PinzBase
import AVFoundation

enum MediaInfoIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"
    case trash = "trash"
}

public struct MediaInfoView: View {

    private let media: MediaItem

    @Environment(\.appRouter) private var router
    @State private var playerController: VideoPlayerController?

    public init(media: MediaItem) {
        self.media = media
    }
    
    public var body: some View {
        CollapsibleHeader(needsBlur: true) {
            header
        } content: {
            ScrollView {
                VStack(spacing: 12) {
                    switch media.type {
                    case .image:
                        LoadableImageThumbnail(url: media.mediaURL) { state in mediaView(for: state) }
                    case .video:
                        videoPlayerSection
                    }

                    privacy

                    delete
                }
            }
            .scrollIndicators(.hidden)
            .padding(.horizontal, 12)
        }
        .onAppear {
            if media.type == .video, let url = media.mediaURL {
                playerController = VideoPlayerController(url: url)
            }
        }
        .onDisappear {
            playerController?.stop()
            playerController = nil
        }
    }

    @ViewBuilder
    private var videoPlayerSection: some View {
        if let controller = playerController {
            let ratio = controller.naturalSize.map { $0.width / $0.height } ?? 1.0
            VideoPlayerView(player: controller.player)
                .aspectRatio(ratio, contentMode: .fit)
                .clipShape(RoundedRectangle(cornerRadius: 36))
                .overlay {
                    if !controller.isPlaying {
                        Image(systemName: "play.fill")
                            .font(.system(size: 52))
                            .foregroundStyle(.white.opacity(0.9))
                            .shadow(radius: 8)
                    }
                }
                .onTapGesture {
                    withAnimation {
                        controller.togglePlayback()
                    }
                }
        }
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
                .cornerRadius(36)
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
                router?.pop(by: 1)
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
