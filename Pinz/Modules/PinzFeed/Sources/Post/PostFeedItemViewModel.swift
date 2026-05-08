import Foundation
import SwiftUI
import MapKit
import PinzDomain
import PinzBase
import PinzNetworking

@MainActor @Observable
final class PostFeedItemViewModel {

    private(set) var post: Post
    private(set) var images: [Int: UIImage] = [:]
    var position: MapCameraPosition

    private let networkService: NetworkServiceProtocol

    enum Intent {
        case like
        case dislike
        case toggleFavourite
    }

    init(
        post: Post,
        networkService: NetworkServiceProtocol = NetworkService.shared
    ) {
        self.post = post
        self.networkService = networkService
        self.position = post.pins.calculateInitialMapPosition(
            zoomMultiplier: 2.5,
            topOffsetFactor: 0.2
        )
    }

    func dispatch(_ intent: Intent) {
        withAnimation {
            switch intent {
            case .like:
                Task { await self.toggleLike() }
            case .dislike:
                Task { await self.toggleDislike() }
            case .toggleFavourite:
                Task { await self.toggleFavourite() }
            }
        }
    }

    func loadImages() async {
        await withTaskGroup(of: (Int, UIImage?).self) { group in
            for index in post.media.indices {
                guard post.media[index].type == .image,
                      let url = post.media[index].mediaURL else { continue }
                group.addTask {
                    let image = await ImageProvider.loadOrGetImage(
                        for: url.absoluteString,
                        .media,
                        cacheVariant: .thumbnail,
                        targetPixel: 420
                    )
                    return (index, image)
                }
            }
            for await (index, image) in group {
                if let image { images[index] = image }
            }
        }
    }

    private func toggleLike() async {
        if post.isLiked {
            post.isLiked = false
            post.likes = max(post.likes - 1, 0)
        } else {
            post.isLiked = true
            post.likes += 1
            if post.isDisliked {
                post.isDisliked = false
                post.dislikes = max(post.dislikes - 1, 0)
            }
        }

        do {
            _ = try await networkService.likeTrip(id: post.id)
        } catch {
            print(error)
        }
    }

    private func toggleDislike() async {
        if post.isDisliked {
            post.isDisliked = false
            post.dislikes = max(post.dislikes - 1, 0)
        } else {
            post.isDisliked = true
            post.dislikes += 1
            if post.isLiked {
                post.isLiked = false
                post.likes = max(post.likes - 1, 0)
            }
        }

        do {
            _ = try await networkService.dislikeTrip(id: post.id)
        } catch {
            print(error)
        }
    }

    private func toggleFavourite() async {
        if post.isSaved {
            post.isSaved = false
            post.favorites = max(post.favorites - 1, 0)
            do {
                try await networkService.removeTripFromFavourites(id: post.id)
            } catch {
                print(error)
            }
        } else {
            post.isSaved = true
            post.favorites += 1
            do {
                _ = try await networkService.addTripToFavourites(id: post.id)
            } catch {
                print(error)
            }
        }
    }
}
