import SwiftUI
import PinzDomain
import AVFoundation

public enum ImageProviderType {
    case group
    case user
    case media

    public var placeholder: UIImage {
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
    private static let logPrefix = "[ImageProvider]"

    public static func loadOrGetImage(
        for urlString: String?,
        _ type: ImageProviderType,
        cacheVariant: MediaCacheVariant = .full,
        targetPixel: Int = 560
    ) async -> UIImage? {
        guard let urlString, let url = URL(string: urlString) else {
            log("[loadOrGetImage] invalid URL")
            return nil
        }

        log("[loadOrGetImage] url=\(urlString) variant=\(cacheVariant.rawValue) type=\(type)")

        if FileManagerImageStorage.shared.isCachingEnabled,
           let cached = FileManagerImageStorage.shared.getImage(url: urlString, variant: cacheVariant) {
            log("[loadOrGetImage] hit cache variant=\(cacheVariant.rawValue) url=\(urlString)")
            return cached
        }

        do {
            log("[loadOrGetImage] miss cache variant=\(cacheVariant.rawValue), downloading url=\(urlString)")
            let (data, _) = try await URLSession.shared.data(from: url)
            if let image = UIImage(data: data) {
                let contentType = imageContentType(for: urlString)
                FileManagerImageStorage.shared.saveImage(
                    image: image,
                    url: urlString,
                    variant: cacheVariant,
                    targetPixel: cacheVariant == .thumbnail ? targetPixel : nil,
                    contentType: contentType
                )
                log(
                    "[loadOrGetImage] saved bytes=\(data.count) " +
                    "variant=\(cacheVariant.rawValue) type=\(type) url=\(urlString)"
                )
                return FileManagerImageStorage.shared.getImage(url: urlString, variant: cacheVariant) ?? image
            }
        } catch {
            log("[loadOrGetImage] failed url=\(urlString) variant=\(cacheVariant.rawValue) error=\(error)")
        }

        return nil
    }

    public static func loadOrGetImage(for urlString: String?, _ type: ImageProviderType) async -> UIImage? {
        await loadOrGetImage(for: urlString, type, cacheVariant: .full)
    }

    public static func loadAndCacheImage(
        for urlString: String,
        _ type: ImageProviderType,
        cacheVariant: MediaCacheVariant = .full,
        targetPixel: Int = 560
    ) async {
        guard FileManagerImageStorage.shared.isCachingEnabled,
              FileManagerImageStorage.shared.getImage(url: urlString, variant: cacheVariant) == nil,
              let url = URL(string: urlString)
        else { return }

        log("[loadAndCacheImage] start url=\(urlString) variant=\(cacheVariant.rawValue) type=\(type)")

        do {
            log("[loadAndCacheImage] miss cache variant=\(cacheVariant.rawValue), downloading url=\(urlString)")
            let (data, _) = try await URLSession.shared.data(from: url)
            if let image = UIImage(data: data) {
                let contentType = imageContentType(for: urlString)
                FileManagerImageStorage.shared.saveImage(
                    image: image,
                    url: urlString,
                    variant: cacheVariant,
                    targetPixel: cacheVariant == .thumbnail ? targetPixel : nil,
                    contentType: contentType
                )
                log("[loadAndCacheImage] saved bytes=\(data.count) variant=\(cacheVariant.rawValue) url=\(urlString)")
            }
        } catch {
            log("[loadAndCacheImage] failed url=\(urlString) variant=\(cacheVariant.rawValue) error=\(error)")
        }
    }

    public static func loadAndCacheImage(for urlString: String, _ type: ImageProviderType) async {
        await loadAndCacheImage(for: urlString, type, cacheVariant: .full)
    }

    public static func loadOrGetImage(
        for urlString: String?,
        type: ImageProviderType,
        cacheVariant: MediaCacheVariant
    ) async -> UIImage? {
        await loadOrGetImage(for: urlString, type, cacheVariant: cacheVariant)
    }
    
    public static func loadOrGetVideoThumbnail(for urlString: String?) async -> UIImage? {
        guard let urlString, let url = URL(string: urlString) else {
            return nil
        }
        
        let cacheKey = "video_thumb_" + urlString
        
        // Проверяем кеш
        if FileManagerImageStorage.shared.isCachingEnabled,
           let cached = FileManagerImageStorage.shared.getImage(url: cacheKey, variant: .thumbnail) {
            log("[loadOrGetVideoThumbnail] hit cache thumbnail url=\(urlString)")
            return cached
        }
        
        // Генерируем thumbnail из видео
        do {
            log("[loadOrGetVideoThumbnail] generate thumbnail url=\(urlString)")
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
            FileManagerImageStorage.shared.saveImage(
                image: uiImage,
                url: cacheKey,
                variant: .thumbnail,
                targetPixel: 420,
                contentType: "image/jpeg"
            )
            log("[loadOrGetVideoThumbnail] generated and saved thumbnail url=\(urlString)")
            return uiImage
        } catch {
            log("[loadOrGetVideoThumbnail] failed url=\(urlString) error=\(error)")
            return nil
        }
    }

    private static func imageContentType(for urlString: String) -> String {
        let trimmed = urlString.lowercased()
        if trimmed.hasSuffix(".png") {
            return "image/png"
        }
        return "image/jpeg"
    }

    private static func log(_ message: String) {
        #if DEBUG
        print("\(logPrefix) \(message)")
        #endif
    }
}
