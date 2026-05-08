import SwiftUI
import CryptoKit

public enum MediaCacheVariant: String, Sendable {
    case thumbnail
    case full
}

protocol LocalFileManagerProtocol {
    func saveImage(image: UIImage, url: String)
    func getImage(url: String) -> UIImage?
    func deleteImage(url: String)
    func clear()
    func getCacheSize() -> String

    func saveImage(image: UIImage, url: String, variant: MediaCacheVariant)
    func getImage(url: String, variant: MediaCacheVariant) -> UIImage?
    func deleteImage(url: String, variant: MediaCacheVariant)
    func clear(variant: MediaCacheVariant)
    func getCacheSize(variant: MediaCacheVariant) -> String
}

public final class FileManagerImageStorage: LocalFileManagerProtocol {
    private static let logPrefix = "[ImageCache]"

    public static let shared = FileManagerImageStorage()
    public static let cacheEnabledKey = "pinz.imageCache.enabled"

    private let legacyFolderName = "rewind-images"
    private let baseFolderName = "rewind-media"
    private let fullSubfolderName = "full"
    private let thumbnailSubfolderName = "thumbnails"
    private let userDefaults = UserDefaults.standard
    private let fileManager = FileManager.default
    private let thumbnailMaxPixel = 560
    private let thumbnailJPEGQuality = CGFloat(0.75)

    private let memoryCaches: [MediaCacheVariant: NSCache<NSString, UIImage>] = [
        .full: NSCache<NSString, UIImage>(),
        .thumbnail: NSCache<NSString, UIImage>()
    ]

    public var isCachingEnabled: Bool {
        get {
            guard let storedValue = userDefaults.object(forKey: Self.cacheEnabledKey) as? Bool else {
                return true
            }
            return storedValue
        }
        set {
            userDefaults.set(newValue, forKey: Self.cacheEnabledKey)
            if !newValue {
                memoryCaches[.full]?.removeAllObjects()
                memoryCaches[.thumbnail]?.removeAllObjects()
            }
        }
    }

    public func saveImage(image: UIImage, url: String) {
        saveImage(image: image, url: url, variant: .full, targetPixel: nil, contentType: nil)
    }

    public func saveImage(image: UIImage, url: String, variant: MediaCacheVariant) {
        saveImage(image: image, url: url, variant: variant, targetPixel: nil, contentType: nil)
    }

    public func getImage(url: String) -> UIImage? {
        getImage(url: url, variant: .full)
    }

    public func deleteImage(url: String) {
        deleteImage(url: url, variant: .full)
    }

    public func clear() {
        clear(variant: .full)
        clear(variant: .thumbnail)
    }

    public func getCacheSize() -> String {
        let fullSize = bytesForFolder(variant: .full)
        let thumbnailSize = bytesForFolder(variant: .thumbnail)
        return formatBytes(fullSize + thumbnailSize)
    }

    public func getImage(url: String, variant: MediaCacheVariant) -> UIImage? {
        #if DEBUG
        print("\(Self.logPrefix) [getImage] start variant=\(variant.rawValue) url=\(url)")
        #endif
        guard isCachingEnabled else {
            return nil
        }

        let imageKey = cacheLookupKey(for: url, variant: variant) as NSString
        if let cachedImage = memoryCaches[variant]?.object(forKey: imageKey) {
            #if DEBUG
            print("\(Self.logPrefix) [getImage] hit variant=\(variant.rawValue) source=memory")
            #endif
            return cachedImage
        }

        if let image = loadImage(for: url, variant: variant) {
            memoryCaches[variant]?.setObject(image, forKey: imageKey)
            #if DEBUG
            print("\(Self.logPrefix) [getImage] hit variant=\(variant.rawValue) source=disk")
            #endif
            return image
        }

        if variant == .full, let legacyImage = getLegacyImage(url: url) {
            let legacyKey = legacyCacheLookupKey(for: url) as NSString
            memoryCaches[.full]?.setObject(legacyImage, forKey: legacyKey)
            memoryCaches[.full]?.setObject(legacyImage, forKey: imageKey)
            #if DEBUG
            print("\(Self.logPrefix) [getImage] hit variant=\(variant.rawValue) source=legacy")
            #endif
            return legacyImage
        }

        #if DEBUG
        print("\(Self.logPrefix) [getImage] miss variant=\(variant.rawValue) url=\(url)")
        #endif
        return nil
    }

    public func deleteImage(url: String, variant: MediaCacheVariant) {
        let normalizedKey = cacheLookupKey(for: url, variant: variant)
        let cacheKey = normalizedKey as NSString
        memoryCaches[variant]?.removeObject(forKey: cacheKey)

        if variant == .full {
            for ext in availableImageExtensions {
                if let url = getImageURL(
                    fileName: normalizedKey,
                    folderName: folderName(for: variant),
                    ext: ext
                ), fileManager.fileExists(atPath: url.path) {
                    try? fileManager.removeItem(at: url)
                }

                if let legacyURL = getImageURL(
                    fileName: legacyFileName(for: url),
                    folderName: legacyFolderName,
                    ext: ext
                ), fileManager.fileExists(atPath: legacyURL.path) {
                    try? fileManager.removeItem(at: legacyURL)
                }
            }
        } else {
            for ext in availableImageExtensions {
                if let url = getImageURL(
                    fileName: normalizedKey,
                    folderName: folderName(for: variant),
                    ext: ext
                ), fileManager.fileExists(atPath: url.path) {
                    try? fileManager.removeItem(at: url)
                }
            }
        }
    }

    public func clear(variant: MediaCacheVariant) {
        memoryCaches[variant]?.removeAllObjects()
        guard let folderURL = getURLForFolder(folderName: folderName(for: variant)),
              fileManager.fileExists(atPath: folderURL.path) else {
            if variant == .full {
                clearLegacyFolder()
            }
            return
        }

        do {
            let fileURLs = try fileManager.contentsOfDirectory(at: folderURL, includingPropertiesForKeys: nil)
            for url in fileURLs {
                try fileManager.removeItem(at: url)
            }
            print("Storage (\(variant.rawValue)) cleared.")
        } catch {
            print("Error clearing storage (\(variant.rawValue)): \(error)")
        }

        if variant == .full {
            clearLegacyFolder()
        }
    }

    public func getCacheSize(variant: MediaCacheVariant) -> String {
        let bytes = bytesForFolder(variant: variant)
        return formatBytes(bytes)
    }

    public func saveImage(
        image: UIImage,
        url: String,
        variant: MediaCacheVariant,
        targetPixel: Int? = nil,
        contentType: String? = nil
    ) {
        let imageSize = "\(Int(image.size.width))x\(Int(image.size.height))"
        guard isCachingEnabled else {
            return
        }

        createFolderIfNeeded(folderName: folderName(for: variant))

        let normalizedContentType = normalizedContentType(
            contentType,
            key: url,
            variant: variant
        )
        let compressionQuality = variant == .thumbnail ? thumbnailJPEGQuality : 1
        let downscaledTargetPixel = variant == .thumbnail ? (targetPixel ?? thumbnailMaxPixel) : targetPixel

        guard let normalizedImage = normalizeAndDownscale(
            image: image,
            maxPixel: downscaledTargetPixel
        ),
        let data = imageData(
            from: normalizedImage,
            contentType: normalizedContentType,
            quality: compressionQuality
        ) else {
            return
        }

        let imageName = cacheLookupKey(for: url, variant: variant)
        let extensionString = fileExtension(for: normalizedContentType)
        guard let fileURL = getImageURL(
            fileName: imageName,
            folderName: folderName(for: variant),
            ext: extensionString
        ) else {
            return
        }

        memoryCaches[variant]?.setObject(normalizedImage, forKey: cacheLookupKey(for: url, variant: variant) as NSString)

        do {
            try data.write(to: fileURL)
            #if DEBUG
            print(
                "\(Self.logPrefix) [saveImage] variant=\(variant.rawValue) " +
                "bytes=\(data.count) size=\(imageSize) " +
                "path=\(fileURL.lastPathComponent)"
            )
            #endif
        } catch let error {
            print("Error saving image: \(imageName). \(error)")
        }
    }

    private func loadImage(for url: String, variant: MediaCacheVariant) -> UIImage? {
        let fileName = cacheLookupKey(for: url, variant: variant)
        for ext in imageExtensions(for: variant) {
            if let fileURL = getImageURL(fileName: fileName, folderName: folderName(for: variant), ext: ext),
               fileManager.fileExists(atPath: fileURL.path),
               let image = UIImage(contentsOfFile: fileURL.path) {
                return image
            }
        }
        return nil
    }

    private func getLegacyImage(url: String) -> UIImage? {
        let fileName = legacyFileName(for: url)
        for ext in availableImageExtensions {
            if let fileURL = getImageURL(fileName: fileName, folderName: legacyFolderName, ext: ext),
               fileManager.fileExists(atPath: fileURL.path),
               let image = UIImage(contentsOfFile: fileURL.path) {
                return image
            }
        }
        return nil
    }

    private var availableImageExtensions: [String] {
        ["png", "jpg", "jpeg"]
    }

    private func imageExtensions(for variant: MediaCacheVariant) -> [String] {
        switch variant {
        case .thumbnail:
            return ["jpg"]
        case .full:
            return availableImageExtensions
        }
    }

    private func bytesForFolder(variant: MediaCacheVariant) -> Int64 {
        let cacheFolder = folderName(for: variant)
        let fileManager = FileManager.default
        var totalBytes: Int64 = 0

        if let folderURL = getURLForFolder(folderName: cacheFolder),
           fileManager.fileExists(atPath: folderURL.path) {
            do {
                let fileURLs = try fileManager.contentsOfDirectory(at: folderURL, includingPropertiesForKeys: [.fileSizeKey])
                for url in fileURLs {
                    let attributes = try fileManager.attributesOfItem(atPath: url.path)
                    totalBytes += (attributes[.size] as? Int64) ?? 0
                }
            } catch {
                return 0
            }
        }

        if variant == .full {
            totalBytes += bytesForLegacyFolder()
        }

        return totalBytes
    }

    private func bytesForLegacyFolder() -> Int64 {
        guard let folderURL = getURLForFolder(folderName: legacyFolderName),
              fileManager.fileExists(atPath: folderURL.path) else {
            return 0
        }

        do {
            let fileURLs = try fileManager.contentsOfDirectory(at: folderURL, includingPropertiesForKeys: [.fileSizeKey])
            return try fileURLs.reduce(into: Int64(0)) { sum, url in
                let attributes = try fileManager.attributesOfItem(atPath: url.path)
                sum += (attributes[.size] as? Int64) ?? 0
            }
        } catch {
            return 0
        }
    }

    private func clearLegacyFolder() {
        guard let folderURL = getURLForFolder(folderName: legacyFolderName),
              fileManager.fileExists(atPath: folderURL.path) else {
            return
        }

        do {
            let fileURLs = try fileManager.contentsOfDirectory(at: folderURL, includingPropertiesForKeys: nil)
            for url in fileURLs {
                try fileManager.removeItem(at: url)
            }
        } catch {
            print("Error clearing legacy storage: \(error)")
        }
    }

    private func cacheLookupKey(for key: String, variant: MediaCacheVariant) -> String {
        return "cache_" + hashString(normalizedImageKey(key) + "::" + variant.rawValue)
    }

    private func legacyCacheLookupKey(for key: String) -> String {
        return "cache_" + normalizedImageKey(key)
    }

    private func legacyFileName(for key: String) -> String {
        return hashString(legacyCacheLookupKey(for: key))
    }

    private func folderName(for variant: MediaCacheVariant) -> String {
        return "\(baseFolderName)/\(variant == .thumbnail ? thumbnailSubfolderName : fullSubfolderName)"
    }

    private func normalizedImageKey(_ key: String) -> String {
        let videoPrefix = "video_thumb_"
        if key.hasPrefix(videoPrefix),
           let parsed = URL(string: String(key.dropFirst(videoPrefix.count))) {
            return "video_thumb_" + pathWithoutQuery(from: parsed)
        }

        if let parsed = URL(string: key) {
            return pathWithoutQuery(from: parsed)
        }

        return key
    }

    private func normalizedContentType(
        _ contentType: String?,
        key: String,
        variant: MediaCacheVariant
    ) -> String {
        if variant == .thumbnail {
            return "image/jpeg"
        }

        if let contentType, contentType.lowercased().contains("png") {
            return "image/png"
        }

        if key.lowercased().hasSuffix(".png") {
            return "image/png"
        }

        return "image/jpeg"
    }

    private func imageData(from image: UIImage, contentType: String, quality: CGFloat) -> Data? {
        if contentType == "image/png" {
            return image.pngData()
        }
        return image.jpegData(compressionQuality: quality)
    }

    private func fileExtension(for contentType: String) -> String {
        contentType == "image/png" ? "png" : "jpg"
    }

    private func normalizeAndDownscale(image: UIImage, maxPixel: Int?) -> UIImage? {
        guard let maxPixel, maxPixel > 0 else {
            return image
        }

        let size = image.size
        let maxDimension = max(size.width, size.height)
        guard maxDimension > CGFloat(maxPixel) else {
            return image
        }

        let scale = CGFloat(maxPixel) / maxDimension
        let targetSize = CGSize(width: size.width * scale, height: size.height * scale)

        let format = UIGraphicsImageRendererFormat()
        let renderer = UIGraphicsImageRenderer(size: targetSize, format: format)
        return renderer.image { _ in
            image.draw(in: CGRect(origin: .zero, size: targetSize))
        }
    }

    private func formatBytes(_ bytes: Int64) -> String {
        let units = ["B", "KB", "MB", "GB"]
        var size = Double(bytes)
        var unitIndex = 0

        while size >= 1024 && unitIndex < units.count - 1 {
            size /= 1024
            unitIndex += 1
        }

        return String(format: "%.1f %@", size, units[unitIndex])
    }
}

extension FileManagerImageStorage {
    private func createFolderIfNeeded(folderName: String) {
        guard let url = getURLForFolder(folderName: folderName) else { return }

        if !fileManager.fileExists(atPath: url.path) {
            do {
                try fileManager.createDirectory(at: url, withIntermediateDirectories: true, attributes: nil)
            } catch let error {
                print("Error creating directory: \(folderName). \(error)")
            }
        }
    }

    private func getURLForFolder(folderName: String) -> URL? {
        guard let url = fileManager.urls(for: .cachesDirectory, in: .userDomainMask).first else {
            return nil
        }
        return url.appendingPathComponent(folderName)
    }

    private func getImageURL(fileName: String, folderName: String, ext: String) -> URL? {
        guard let folderURL = getURLForFolder(folderName: folderName) else {
            return nil
        }
        return folderURL.appendingPathComponent(fileName).appendingPathExtension(ext)
    }

    private func pathWithoutQuery(from url: URL) -> String {
        var components = URLComponents(url: url, resolvingAgainstBaseURL: false)
        components?.query = nil
        components?.fragment = nil
        return components?.url?.absoluteString ?? url.absoluteString
    }

    private func hashString(_ text: String) -> String {
        let digest = SHA256.hash(data: Data(text.utf8))
        return digest.map { String(format: "%02x", $0) }.joined()
    }
}
