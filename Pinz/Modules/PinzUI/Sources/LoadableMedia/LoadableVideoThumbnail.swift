import Foundation
import AVFoundation
import SwiftUI
import PinzBase

public struct LoadableVideoThumbnail<Content: View>: View {
    let url: URL?
    @ViewBuilder let content: (LoadableMediaState) -> Content
    @State private var state: LoadableMediaState = .empty

    public init(
        url: URL? = nil,
        content: @escaping (LoadableMediaState) -> Content
    ) {
        self.url = url
        self.content = content
    }

    public var body: some View {
        content(state)
            .task(id: url) {
                await loadThumbnail(url)
            }
    }

    private func loadThumbnail(_ currentURL: URL?) async {
        animateState(to: .empty)
        guard let url = currentURL else {
            animateState(to: .failure)
            return
        }
        
        if let thumbnail = await ImageProvider.loadOrGetVideoThumbnail(for: url.absoluteString) {
            animateState(to: .ready(thumbnail))
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
