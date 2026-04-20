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

    private enum Source {
        case remote(MediaItem)
        case local(LoadedMedia)
    }

    private let source: Source

    @Environment(\.appRouter) private var router
    @State private var playerController: VideoPlayerController?

    public init(media: MediaItem) {
        self.source = .remote(media)
    }

    public init(localMedia: LoadedMedia) {
        self.source = .local(localMedia)
    }

    public var body: some View {
        CollapsibleHeader(needsBlur: true) {
            header
        } content: {
            ScrollView {
                VStack(spacing: 12) {
                    mediaSection

                    privacy

                    delete
                }
            }
            .scrollIndicators(.hidden)
            .padding(.horizontal, 12)
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { setupPlayer() }
        .onDisappear {
            playerController?.stop()
            playerController = nil
        }
    }

    private func setupPlayer() {
        switch source {
        case .remote(let media):
            if media.type == .video, let url = media.mediaURL {
                playerController = VideoPlayerController(url: url)
            }
        case .local(let media):
            if case .video(let url, _) = media.content {
                playerController = VideoPlayerController(url: url)
            }
        }
    }

    @ViewBuilder
    private var mediaSection: some View {
        switch source {
        case .remote(let media):
            switch media.type {
            case .image:
                LoadableImageThumbnail(url: media.mediaURL) { state in mediaView(for: state) }
            case .video:
                videoPlayerSection
            }
        case .local(let media):
            switch media.content {
            case .image(let image):
                CollapsibleImageView(image: image)
            case .video:
                videoPlayerSection
            case .loading:
                Rectangle()
                    .fill(Color.gray.opacity(0.3))
                    .aspectRatio(1, contentMode: .fit)
                    .cornerRadius(24)
                    .overlay { ProgressView().tint(.white) }
            }
        }
    }

    @ViewBuilder
    private var videoPlayerSection: some View {
        if let controller = playerController {
            let ratio = controller.naturalSize.map { $0.width / $0.height } ?? 1.0
            CollapsibleView {
                VideoPlayerView(player: controller.player)
                    .aspectRatio(ratio, contentMode: .fill)
            }
            .clipShape(RoundedRectangle(cornerRadius: 24))
            .overlay {
                if !controller.isPlaying {
                    Image(systemName: "play.fill")
                        .font(.system(size: 52))
                        .foregroundStyle(.white.opacity(0.9))
                        .shadow(radius: 8)
                }

                VStack {
                    HStack {
                        BadgeView(
                            icon: controller.isMuted ? .soundOff : .soundOn,
                            badgeSize: 36,
                            iconSize: 18
                        ) {
                            controller.toggleMute()
                        }.disabledWithOpacity(!controller.hasAudio, opacity: 0.5)
                        Spacer()
                    }
                    Spacer()
                }.padding(8)
            }
            .onTapGesture {
                withAnimation(.easeInOut(duration: 0.3)) {
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
                .cornerRadius(24)
        case .ready(let image):
            CollapsibleImageView(image: image)
        case .failure:
            Rectangle()
                .fill(Color.gray.opacity(0.3))
                .aspectRatio(1, contentMode: .fit)
                .overlay {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundColor(.white)
                }
                .cornerRadius(24)
        }
    }

    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { router?.pop(by: 1) }
            )
        }, rightView: {
            PinzButton(
                type: .icon(.crop),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { }
            )
            PinzButton(
                type: .icon(.download),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { }
            )
        })
    }

    private var privacy: some View {
        PrivacySection(members: TripMember.stubs())
    }

    private var delete: some View {
        SettingsGroup(settings: [
            .default(Setting.DefaultSetting(
                id: "mediaDelete",
                leading: .iconTitle(MediaInfoIcon.trash, PinzBaseStrings.MediaInfo.Button.delete),
                trailing: .icon(MediaInfoIcon.chevronRight),
                style: .destructive,
                action: .plain { }
            ))
        ])
    }
}
