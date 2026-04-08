import Foundation
import SwiftUI
import MapKit
import PinzDomain
import PinzBase

@MainActor @Observable
final class PostViewModel {

    private(set) var post: Post
    private(set) var images: [Int: UIImage] = [:]
    var position: MapCameraPosition

    enum Intent {
        // extend as needed
    }

    init(post: Post) {
        self.post = post
        self.position = post.pins.calculateInitialMapPosition(zoomMultiplier: 2.5, topOffsetFactor: 0.2)
    }

    func dispatch(_ intent: Intent) {
        switch intent { }
    }

    func loadImages() async {
        await withTaskGroup(of: (Int, UIImage?).self) { group in
            for index in post.pins.indices {
                guard let media = post.pins[index].medias.filter({ $0.type == .image }).randomElement(),
                      let url = media.mediaURL else { continue }
                group.addTask {
                    let image = await ImageProvider.loadOrGetImage(for: url.absoluteString, .media)
                    return (index, image)
                }
            }
            for await (index, image) in group {
                if let image { images[index] = image }
            }
        }
    }
}
