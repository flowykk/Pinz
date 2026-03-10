import SwiftUI
import AVFoundation

public struct VideoPlayerView: UIViewRepresentable {
    let player: AVPlayer

    public init(player: AVPlayer) {
        self.player = player
    }

    public func makeUIView(context: Context) -> PlayerUIView {
        PlayerUIView(player: player)
    }

    public func updateUIView(_ uiView: PlayerUIView, context: Context) {
        uiView.playerLayer.player = player
    }
}

public final class PlayerUIView: UIView {
    public let playerLayer = AVPlayerLayer()

    init(player: AVPlayer) {
        super.init(frame: .zero)
        playerLayer.player = player
        playerLayer.videoGravity = .resizeAspect
        layer.addSublayer(playerLayer)
    }

    required init?(coder: NSCoder) { fatalError() }

    public override func layoutSubviews() {
        super.layoutSubviews()
        playerLayer.frame = bounds
    }
}
