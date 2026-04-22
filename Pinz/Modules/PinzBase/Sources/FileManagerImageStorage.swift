import SwiftUI
import CryptoKit

protocol LocalFileManagerProtocol {
    func saveImage(image: UIImage, url: String)
    func getImage(url: String) -> UIImage?
    func deleteImage(url: String)
    func clear()
}

public final class FileManagerImageStorage: LocalFileManagerProtocol {
    public static let shared = FileManagerImageStorage()
    public static let cacheEnabledKey = "pinz.imageCache.enabled"

    private let folderName = "rewind-images"
    private let memoryCache = NSCache<NSString, UIImage>()
    private let userDefaults = UserDefaults.standard

    private init() { }

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
                memoryCache.removeAllObjects()
            }
        }
    }

    public func saveImage(image: UIImage, url: String) {
        print(#function)
        guard isCachingEnabled else {
            return
        }

        let imageName = fileName(for: url)
        createFolderIfNeeded(folderName: folderName)

        guard let data = image.pngData(),
              let fileURL = getURLForImage(urlToArticle: imageName, folderName: folderName)
        else { return }

        memoryCache.setObject(image, forKey: cacheKey(for: url) as NSString)

        do {
            print("SAVING IMAGE")
            try data.write(to: fileURL)
            print("SAVED IMAGE")
        } catch let error {
            print("Error saving image: \(imageName). \(error)")
        }
    }

    public func getImage(url: String) -> UIImage? {
        print(#function)
        guard isCachingEnabled else {
            return nil
        }

        let imageKey = cacheKey(for: url) as NSString
        if let cachedImage = memoryCache.object(forKey: imageKey) {
            print("RETURNING IMAGE FROM CACHE")
            return cachedImage
        }

        let imageName = fileName(for: url)
        guard let url = getURLForImage(urlToArticle: imageName, folderName: folderName),
              FileManager.default.fileExists(atPath: url.path)
        else {
            return nil
        }

        if let image = UIImage(contentsOfFile: url.path) {
            memoryCache.setObject(image, forKey: imageKey)
            print("RETURNING IMAGE FROM DISK")
            return image
        }

        return nil
    }

    public func deleteImage(url: String) {
        let imageName = fileName(for: url)
        let cacheKey = cacheKey(for: url) as NSString
        memoryCache.removeObject(forKey: cacheKey)
        guard let url = getURLForImage(urlToArticle: imageName, folderName: folderName),
            FileManager.default.fileExists(atPath: url.path) else {
            return
        }

        do {
            try FileManager.default.removeItem(at: url)
        } catch let error {
            print("Error deleting image: \(folderName). \(error)")
        }
    }

    public func clear() {
        guard let folderURL = getURLForFolder(folderName: folderName),
              FileManager.default.fileExists(atPath: folderURL.path) else {
            return
        }

        do {
            let fileURLs = try FileManager.default.contentsOfDirectory(at: folderURL, includingPropertiesForKeys: nil)
            for url in fileURLs {
                try FileManager.default.removeItem(at: url)
            }
            memoryCache.removeAllObjects()
            print("Storage cleared.")
        } catch {
            print("Error clearing storage: \(error)")
        }
    }

    public func getCacheSize() -> String {
        guard let folderURL = getURLForFolder(folderName: folderName) else { return "0 B" }
        let fileManager = FileManager.default
        guard fileManager.fileExists(atPath: folderURL.path) else { return "0 B" }

        do {
            let fileURLs = try fileManager.contentsOfDirectory(at: folderURL, includingPropertiesForKeys: [.fileSizeKey])
            let size: Int64 = try fileURLs.reduce(0) { sum, url in
                let attributes = try fileManager.attributesOfItem(atPath: url.path)
                return sum + (attributes[.size] as? Int64 ?? 0)
            }
            return formatBytes(size)
        } catch {
            return "0 B"
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

        if !FileManager.default.fileExists(atPath: url.path) {
            do {
                try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true, attributes: nil)
            } catch let error {
                print("Error creating directory: \(folderName). \(error)")
            }
        }
    }

    private func getURLForFolder(folderName: String) -> URL? {
        guard let url = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first else {
            return nil
        }
        return url.appendingPathComponent(folderName)
    }

    private func getURLForImage(urlToArticle: String, folderName: String) -> URL? {
        guard let folderURL = getURLForFolder(folderName: folderName) else {
            return nil
        }
        return folderURL.appendingPathComponent(urlToArticle + ".png")
    }

    private func cacheKey(for key: String) -> String {
        return "cache_" + normalizedImageKey(key)
    }

    private func fileName(for key: String) -> String {
        return hashString(cacheKey(for: key))
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
