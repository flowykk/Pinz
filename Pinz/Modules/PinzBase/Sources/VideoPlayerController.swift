import AVFoundation

@MainActor @Observable
public final class VideoPlayerController {
    public let player: AVPlayer
    public private(set) var isPlaying: Bool = false
    public private(set) var naturalSize: CGSize?
    public private(set) var isMuted: Bool = false
    public private(set) var hasAudio: Bool = false

    public init(url: URL) {
        self.player = AVPlayer(url: url)
        Task {
            async let thumbnail = ImageProvider.loadOrGetVideoThumbnail(for: url.absoluteString)
            async let tracks = AVURLAsset(url: url).load(.tracks)

            if let image = await thumbnail {
                self.naturalSize = image.size
            }
            if let loaded = try? await tracks {
                self.hasAudio = loaded.contains(where: { $0.mediaType == .audio })
            }
        }
        NotificationCenter.default.addObserver(
            self,
            selector: #selector(playerDidFinish),
            name: .AVPlayerItemDidPlayToEndTime,
            object: player.currentItem
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
            activateAudioSession()
            player.play()
        }
        isPlaying.toggle()
    }

    public func toggleMute() {
        isMuted.toggle()
        player.isMuted = isMuted
    }

    private func activateAudioSession() {
        let session = AVAudioSession.sharedInstance()
        try? session.setCategory(.playback, mode: .default)
        try? session.setActive(true)
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
