import Foundation
import CryptoKit

public actor VideoFileStorage {
    public static let shared = VideoFileStorage()
    private static let logPrefix = "[VideoCache]"

    private let folderName = "rewind-media/videos/full"
    private let fileManager = FileManager.default

    public func cachedVideoURL(for remoteURL: URL) -> URL? {
        let cacheFileURL = cacheFileURL(for: remoteURL)
        #if DEBUG
        print(
            "\(Self.logPrefix) [cachedVideoURL] check " +
            "url=\(remoteURL.absoluteString) file=\(cacheFileURL.lastPathComponent)"
        )
        #endif
        guard fileManager.fileExists(atPath: cacheFileURL.path) else {
            return nil
        }
        return cacheFileURL
    }

    public func cacheVideoFullIfNeeded(from remoteURL: URL) async -> URL {
        #if DEBUG
        print("\(Self.logPrefix) [cacheVideoFullIfNeeded] request url=\(remoteURL.absoluteString)")
        #endif
        guard FileManagerImageStorage.shared.isCachingEnabled else {
            return remoteURL
        }
        if remoteURL.isFileURL {
            #if DEBUG
            print("\(Self.logPrefix) [cacheVideoFullIfNeeded] skip local url=\(remoteURL.absoluteString)")
            #endif
            return remoteURL
        }

        if let cached = cachedVideoURL(for: remoteURL) {
            #if DEBUG
            print("\(Self.logPrefix) [cacheVideoFullIfNeeded] hit=\(cached.lastPathComponent)")
            #endif
            return cached
        }

        let destination = cacheFileURL(for: remoteURL)
        #if DEBUG
        print("\(Self.logPrefix) [cacheVideoFullIfNeeded] miss, destination=\(destination.lastPathComponent)")
        #endif
        do {
            try fileManager.createDirectory(
                at: getFolderURL(),
                withIntermediateDirectories: true,
                attributes: nil
            )
            let (downloadedURL, _) = try await URLSession.shared.download(from: remoteURL)
            try? fileManager.removeItem(at: destination)
            try fileManager.moveItem(at: downloadedURL, to: destination)
            #if DEBUG
            print("\(Self.logPrefix) [cacheVideoFullIfNeeded] downloaded to=\(destination.lastPathComponent)")
            #endif
            return destination
        } catch {
            print("Failed to cache full video for \(remoteURL): \(error)")
        }

        return remoteURL
    }

    public func clear() {
        let folderURL = getFolderURL()
        guard fileManager.fileExists(atPath: folderURL.path) else {
            return
        }

        do {
            let fileURLs = try fileManager.contentsOfDirectory(at: folderURL, includingPropertiesForKeys: nil)
            for url in fileURLs {
                try fileManager.removeItem(at: url)
            }
        } catch {
            print("Failed to clear cached videos: \(error)")
        }
    }

    public func getCacheSize() -> String {
        let folderURL = getFolderURL()
        guard fileManager.fileExists(atPath: folderURL.path) else {
            return "0 B"
        }

        do {
            let fileURLs = try fileManager.contentsOfDirectory(at: folderURL, includingPropertiesForKeys: [.fileSizeKey])
            let size = try fileURLs.reduce(Int64(0)) { sum, url in
                let attributes = try fileManager.attributesOfItem(atPath: url.path)
                return sum + (attributes[.size] as? Int64 ?? 0)
            }
            return formatBytes(size)
        } catch {
            return "0 B"
        }
    }

    private func cacheFileURL(for remoteURL: URL) -> URL {
        let fileName = hashString(normalizedKey(for: remoteURL))
        let fileExtension = remoteURL.pathExtension.isEmpty ? "mp4" : remoteURL.pathExtension
        return getFolderURL().appendingPathComponent(fileName).appendingPathExtension(fileExtension)
    }

    private func getFolderURL() -> URL {
        guard let caches = fileManager.urls(for: .cachesDirectory, in: .userDomainMask).first else {
            return URL(fileURLWithPath: NSTemporaryDirectory())
        }
        return caches.appendingPathComponent(folderName)
    }

    private func normalizedKey(for remoteURL: URL) -> String {
        var components = URLComponents(url: remoteURL, resolvingAgainstBaseURL: false)
        components?.query = nil
        components?.fragment = nil
        return components?.url?.absoluteString ?? remoteURL.absoluteString
    }

    private func hashString(_ text: String) -> String {
        let digest = SHA256.hash(data: Data(text.utf8))
        return digest.map { String(format: "%02x", $0) }.joined()
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
