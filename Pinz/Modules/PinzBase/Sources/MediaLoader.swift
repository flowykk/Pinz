import SwiftUI
import AVFoundation
import PhotosUI
import UniformTypeIdentifiers
import PinzDomain

public final class MediaLoader {
    public static let shared = MediaLoader()
    private init() {}

    public func load(from item: PhotosPickerItem, id: UUID = UUID()) async -> LoadedMedia? {
        if isVideoItem(item) {
            guard let transfer = try? await item.loadTransferable(type: VideoFileTransferable.self),
                  let frame = await firstFrame(from: transfer.url) else { return nil }
            return LoadedMedia(id: id, content: .video(url: transfer.url, firstFrame: frame), photosPickerItem: item)
        } else {
            guard let data = try? await item.loadTransferable(type: Data.self),
                  let image = UIImage(data: data) else { return nil }
            return LoadedMedia(id: id, content: .image(image), photosPickerItem: item)
        }
    }

    public func firstFrame(from url: URL) async -> UIImage? {
        let asset = AVURLAsset(url: url)
        let generator = AVAssetImageGenerator(asset: asset)
        generator.appliesPreferredTrackTransform = true
        guard let cgImage = try? await generator.image(at: .zero).image else { return nil }
        return UIImage(cgImage: cgImage)
    }

    private func isVideoItem(_ item: PhotosPickerItem) -> Bool {
        item.supportedContentTypes.contains { $0.conforms(to: .audiovisualContent) }
    }
}

private struct VideoFileTransferable: Transferable {
    let url: URL

    static var transferRepresentation: some TransferRepresentation {
        FileRepresentation(contentType: .movie) { video in
            SentTransferredFile(video.url)
        } importing: { received in
            let dest = FileManager.default.temporaryDirectory
                .appendingPathComponent(UUID().uuidString + ".mov")
            try FileManager.default.copyItem(at: received.file, to: dest)
            return VideoFileTransferable(url: dest)
        }
    }
}
