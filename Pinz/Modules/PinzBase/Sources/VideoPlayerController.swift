import AVFoundation

@MainActor @Observable
public final class VideoPlayerController {
    public let player: AVPlayer
    public private(set) var isPlaying: Bool = false
    public private(set) var naturalSize: CGSize?

    public init(url: URL) {
        self.player = AVPlayer(url: url)
        Task {
            if let thumbnail = await ImageProvider.loadOrGetVideoThumbnail(for: url.absoluteString) {
                self.naturalSize = thumbnail.size
            }
        }
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(playerDidFinish),
            name: .AVPlayerItemDidPlayToEndTime,
            object: nil
        )
    }

    @objc private func playerDidFinish() {
        player.seek(to: .zero)
        player.play()
    }

    public func togglePlayback() {
        if isPlaying {
            player.pause()
        } else {
            player.play()
        }
        isPlaying.toggle()
    }

    public func stop() {
        player.pause()
        player.seek(to: .zero)
        isPlaying = false
    }

    deinit {
        NotificationCenter.default.removeObserver(self)
    }
}
