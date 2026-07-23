import SwiftUI
import PinzDomain
import PinzUI
import PinzBase
import AVFoundation

public struct PhotoBattleView: View {

    @Bindable var viewModel: PhotoBattleViewModel
    @State private var leftPlayerController: VideoPlayerController?
    @State private var rightPlayerController: VideoPlayerController?
    @State private var winnerPlayerController: VideoPlayerController?
    private let battleMediaHeight = UIScreen.main.bounds.height / 2
    private let battleMediaWidth = UIScreen.main.bounds.width / 2

    init(viewModel: PhotoBattleViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        ZStack {
            LinearGradient(
                colors: [
                    Color(red: 8 / 255, green: 8 / 255, blue: 10 / 255),
                    Color(red: 20 / 255, green: 20 / 255, blue: 30 / 255)
                ],
                startPoint: .top,
                endPoint: .bottom
            )
            .ignoresSafeArea()

            VStack(spacing: 14) {
                topBar

                if viewModel.battleMode == .battle {
                    if viewModel.battleError == nil {
                        progressPanel
                        if viewModel.isPhotoBattlePreloading {
                            HStack(spacing: 8) {
                                ProgressView()
                                    .tint(.white)
                                Text(PinzBaseStrings.PhotoBattle.loadingMedias)
                                    .roundedFont(size: 12, foregroundColor: .white.opacity(0.8))
                            }
                            .transition(.opacity.combined(with: .scale(scale: 0.96)))
                            .animation(.easeInOut(duration: 0.25), value: viewModel.isPhotoBattlePreloading)
                        }
                        Divider().overlay(.white.opacity(0.2))
                    }
                } else {
                    winnerPanel
                }

                if let error = viewModel.battleError {
                    errorBanner(message: error)
                }

                if viewModel.battleMode == .battle {
                    battlePanels

                    if viewModel.leftMedia != nil, viewModel.rightMedia != nil {
                        choiceButtons
                    } else if !viewModel.isSubmittingResult {
                        ProgressView()
                            .tint(.white)
                    }

                    if viewModel.isSubmittingResult {
                        VStack(spacing: 8) {
                            ProgressView()
                                .tint(.white)
                            Text(PinzBaseStrings.PhotoBattle.loadingSaving)
                                .roundedFont(size: 14, foregroundColor: .white.opacity(0.8))
                        }
                    }
                } else {
                Button {
                    viewModel.dismissPhotoBattle()
                } label: {
                    Text(PinzBaseStrings.PhotoBattle.Footer.close)
                        .roundedFont(size: 15, weight: .semibold, foregroundColor: .white)
                        .frame(maxWidth: .infinity)
                        .frame(height: 52)
                            .background(.white.opacity(0.18))
                            .clipShape(Capsule())
                    }
                    .buttonStyle(.plain)
                }

                Spacer()
            }
            .padding(.horizontal, 14)
            .padding(.top, 10)
        }
    }

    private var topBar: some View {
        HStack {
            PinzButton(
                type: .icon(.xmark),
                tint: .white,
                action: .plain { viewModel.dismissPhotoBattle() }
            )

            Spacer()

            VStack(alignment: .center) {
                Text(PinzBaseStrings.PhotoBattle.title)
                    .roundedFont(size: 18, weight: .bold, foregroundColor: .white)

                if viewModel.battleMode == .battle {
                    Text(PinzBaseStrings.PhotoBattle.Progress.round(
                        viewModel.currentRound,
                        PhotoBattleViewModel.totalBattleRounds,
                        viewModel.step,
                        viewModel.totalBattleSteps
                    ))
                    .roundedFont(size: 13, foregroundColor: .white.opacity(0.8))
                }
            }

            Spacer()
            Text("")
                .frame(width: 40)
        }
        .padding(.top, 4)
    }

    private var progressPanel: some View {
        VStack(alignment: .leading, spacing: 10) {
            ProgressView(value: viewModel.progress)
                .tint(.white)
                .animation(.easeInOut(duration: 0.2), value: viewModel.progress)
            Text(PinzBaseStrings.PhotoBattle.Progress.comparisons(viewModel.step, viewModel.totalBattleSteps))
                .roundedFont(size: 12, foregroundColor: .white.opacity(0.8))
        }
    }

    private var winnerPanel: some View {
        VStack(spacing: 12) {
            Text(PinzBaseStrings.PhotoBattle.Winner.title)
                .roundedFont(size: 20, weight: .bold, foregroundColor: .white)

            BattleMediaPanel(
                media: viewModel.winnerPhotoBattleMedia,
                isLocked: true,
                playerController: $winnerPlayerController
            )
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .frame(height: battleMediaHeight)
            .clipShape(RoundedRectangle(cornerRadius: 16))

            Text(viewModel.winnerBattleRating.map(PinzBaseStrings.PhotoBattle.Winner.rating) ?? PinzBaseStrings.PhotoBattle.Winner.loadingRating)
                .roundedFont(size: 14, foregroundColor: .white.opacity(0.9))
                .multilineTextAlignment(.center)
        }
    }

    private var battlePanels: some View {
        HStack(spacing: 12) {
            battleMediaPanel(viewModel.leftMedia, isLeft: true)
            battleMediaPanel(viewModel.rightMedia, isLeft: false)
        }
        .frame(height: battleMediaHeight)
    }

    private var choiceButtons: some View {
        @ViewBuilder func button(for media: PhotoBattleMedia) -> some View {
            Button {
                viewModel.selectPhotoBattleMedia(media)
            } label: {
                Text(PinzBaseStrings.PhotoBattle.Button.chooseMedia)
                    .roundedFont(size: 15, weight: .semibold, foregroundColor: .white)
                    .frame(maxWidth: .infinity)
                    .frame(height: 52)
                    .background(.white.opacity(0.18))
                    .clipShape(Capsule())
            }
            .disabled(viewModel.isBattleControlsBlocked)
            .buttonStyle(.plain)
        }

        return HStack(spacing: 12) {
            if let left = viewModel.leftMedia {
                button(for: left)
            }

            if let right = viewModel.rightMedia {
                button(for: right)
            }
        }
    }

    private func battleMediaPanel(_ media: PhotoBattleMedia?, isLeft: Bool) -> some View {
        BattleMediaPanel(
            media: media,
            isLocked: viewModel.isBattleControlsBlocked,
            playerController: isLeft ? $leftPlayerController : $rightPlayerController
        )
            .frame(maxWidth: .infinity, maxHeight: .infinity)
            .frame(width: battleMediaWidth - 6, height: battleMediaHeight)
        .clipShape(RoundedRectangle(cornerRadius: 16))
            .id(isLeft ? "battle-left-panel-\(media?.photoBattleMediaId ?? "empty")" : "battle-right-panel-\(media?.photoBattleMediaId ?? "empty")")
    }

    private func errorBanner(message: String) -> some View {
        VStack(spacing: 6) {
            Text(message)
                .roundedFont(size: 13, foregroundColor: .white)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
                .background(Color.red.opacity(0.35))
                .cornerRadius(12)

            Button {
                viewModel.clearPhotoBattleError()
            } label: {
                Text(PinzBaseStrings.PhotoBattle.Footer.close)
                    .roundedFont(size: 14, foregroundColor: .white)
            }
            .buttonStyle(.plain)
        }
    }
}

private struct BattleMediaPanel: View {
    let media: PhotoBattleMedia?
    let isLocked: Bool
    @Binding var playerController: VideoPlayerController?

    var body: some View {
        Group {
            if let media {
                ZStack {
                    if media.kind == .video {
                        if let playerController {
                            BattleVideoPlayerView(player: playerController.player)
                                .allowsHitTesting(false)
                                .clipped()
                        } else {
                            MediaThumbnailView(
                                url: media.url,
                                type: .video,
                                contentMode: .fill,
                                cornerRadius: 16
                            )
                        }
                    } else {
                        MediaThumbnailView(
                            url: media.url,
                            type: media.kind,
                            contentMode: .fill,
                            cornerRadius: 16
                        )
                    }

                    if media.kind == .video {
                        let isPlaying = playerController?.isPlaying ?? false

                        if !isPlaying {
                            Image(systemName: "play.fill")
                                .font(.system(size: 52))
                                .foregroundStyle(.white.opacity(0.9))
                                .shadow(radius: 8)
                        }
                    }

                    if isLocked {
                        Color.black.opacity(0.28)
                    }
                }
            } else {
                Rectangle()
                    .fill(Color.white.opacity(0.15))
                    .overlay {
                        ProgressView()
                            .tint(.white)
                    }
            }
        }
        .contentShape(RoundedRectangle(cornerRadius: 16))
        .onTapGesture {
            guard !isLocked, media?.kind == .video else { return }
            withAnimation(.easeInOut(duration: 0.3)) {
                playerController?.togglePlayback()
            }
        }
        .cornerRadius(16)
        .onAppear {
            configurePlayerController(for: media)
        }
        .onChange(of: media?.photoBattleMediaId) { _, _ in
            configurePlayerController(for: media)
        }
        .onDisappear { resetController() }
    }

    private func configurePlayerController(for media: PhotoBattleMedia?) {
        guard let media, media.kind == .video, let url = media.url else {
            playerController?.stop()
            playerController = nil
            return
        }

        playerController?.stop()
        Task {
            let cachedURL = await VideoFileStorage.shared.cacheVideoFullIfNeeded(from: url)
            await MainActor.run {
                playerController = VideoPlayerController(url: cachedURL)
            }
        }
    }

    private func resetController() {
        playerController?.stop()
        playerController = nil
    }
}

private struct BattleVideoPlayerView: UIViewRepresentable {
    let player: AVPlayer

    func makeUIView(context: Context) -> PlayerUIView {
        PlayerUIView(player: player)
    }

    func updateUIView(_ uiView: PlayerUIView, context: Context) {
        uiView.playerLayer.player = player
        uiView.playerLayer.videoGravity = .resizeAspectFill
    }

    final class PlayerUIView: UIView {
        let playerLayer = AVPlayerLayer()

        init(player: AVPlayer) {
            super.init(frame: .zero)
            playerLayer.player = player
            playerLayer.videoGravity = .resizeAspectFill
            layer.addSublayer(playerLayer)
        }

        required init?(coder: NSCoder) { fatalError() }

        override func layoutSubviews() {
            super.layoutSubviews()
            playerLayer.frame = bounds
        }
    }
}
