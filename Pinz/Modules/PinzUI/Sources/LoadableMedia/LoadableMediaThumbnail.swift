import Foundation
import SwiftUI
import PinzBase

public struct LoadableImageThumbnail<Content: View>: View {
    private var url: URL?
    @ViewBuilder let content: (LoadableMediaState) -> Content
    private let cacheVariant: MediaCacheVariant
    private let cacheTargetPixel: Int
    @State private var state: LoadableMediaState = .empty

    public init(
        url: URL? = nil,
        cacheVariant: MediaCacheVariant = .thumbnail,
        cacheTargetPixel: Int = 560,
        content: @escaping (LoadableMediaState) -> Content
    ) {
        self.url = url
        self.cacheVariant = cacheVariant
        self.cacheTargetPixel = cacheTargetPixel
        self.content = content
    }

    public var body: some View {
        content(state)
            .onChange(of: url) { _, newValue in
                Task { await loadImage(newValue) }
            }
            .task {
                await loadImage(url)
            }
    }

    private func loadImage(_ currentURL: URL?) async {
        animateState(to: .empty)
        guard let url = currentURL else {
            animateState(to: .failure)
            return
        }
        await withTaskGroup(of: Void.self) { group in
            group.addTask {
                await ImageProvider.loadAndCacheImage(
                    for: url.absoluteString,
                    .media,
                    cacheVariant: cacheVariant,
                    targetPixel: cacheTargetPixel
                )
            }
        }
        if let uiImage = await ImageProvider.loadOrGetImage(
            for: url.absoluteString,
            .media,
            cacheVariant: cacheVariant,
            targetPixel: cacheTargetPixel
        ) {
            animateState(to: .ready(uiImage))
        } else {
            animateState(to: .failure)
        }
    }

    private func animateState(to state: LoadableMediaState) {
        withAnimation {
            self.state = state
        }
    }
}
