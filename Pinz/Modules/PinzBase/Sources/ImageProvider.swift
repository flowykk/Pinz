import SwiftUI
import PinzDomain
import AVFoundation

public enum ImageProviderType {
    case group
    case user
    case media

    var placeholder: UIImage {
        switch self {
        case .group:
            return PinzDomainAsset.groupPlaceholder.image
        case .user:
            return PinzDomainAsset.userPlacholder.image
        case .media:
            return PinzDomainAsset.defaultPlaceholder.image
        }
    }
}

public enum ImageProvider {
    public static func loadOrGetImage(for urlString: String?, _ type: ImageProviderType) async -> UIImage? {
        guard let urlString, let url = URL(string: urlString) else {
            return nil
        }

        if let cached = FileManagerImageStorage.shared.getImage(url: urlString) {
            return cached
        }

        do {
            let (data, _) = try await URLSession.shared.data(from: url)
            if let image = UIImage(data: data) {
                FileManagerImageStorage.shared.saveImage(image: image, url: urlString)
                return image
            }
        } catch {
            print("(loadOrGetImage) Failed to load image for \(urlString): \(error)")
        }

        return nil
    }

    public static func loadAndCacheImage(for urlString: String, _ type: ImageProviderType) async {
        guard FileManagerImageStorage.shared.getImage(url: urlString) == nil,
              let url = URL(string: urlString)
        else { return }

        do {
            let (data, _) = try await URLSession.shared.data(from: url)
            if let image = UIImage(data: data) {
                FileManagerImageStorage.shared.saveImage(image: image, url: urlString)
            }
        } catch {
            print("(loadAndCacheImage) Failed to load image for \(urlString): \(error)")
        }
    }
    
    public static func loadOrGetVideoThumbnail(for urlString: String?) async -> UIImage? {
        guard let urlString, let url = URL(string: urlString) else {
            return nil
        }
        
        let cacheKey = "video_thumb_" + urlString
        
        // Проверяем кеш
        if let cached = FileManagerImageStorage.shared.getImage(url: cacheKey) {
            return cached
        }
        
        // Генерируем thumbnail из видео
        do {
            let asset = AVURLAsset(url: url)
            let imageGenerator = AVAssetImageGenerator(asset: asset)
            imageGenerator.appliesPreferredTrackTransform = true
            let time = CMTime(seconds: 0.0, preferredTimescale: 600)
            
            let cgImage = try await withCheckedThrowingContinuation { (cont: CheckedContinuation<CGImage, Error>) in
                imageGenerator.generateCGImagesAsynchronously(
                    forTimes: [NSValue(time: time)]
                ) { _, cgImage, _, result, error in
                    if let cgImage, result == .succeeded {
                        cont.resume(returning: cgImage)
                    } else if let error {
                        cont.resume(throwing: error)
                    } else {
                        cont.resume(throwing: NSError(domain: "VideoThumbnail", code: -1))
                    }
                }
            }
            
            let uiImage = UIImage(cgImage: cgImage)
            FileManagerImageStorage.shared.saveImage(image: uiImage, url: cacheKey)
            return uiImage
        } catch {
            print("(loadOrGetVideoThumbnail) Failed to generate video thumbnail for \(urlString): \(error)")
            return nil
        }
    }
}
