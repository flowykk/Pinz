import AVFoundation
import Foundation
import UIKit

public enum MediaUploadKind: String {
    case image
    case video
}

public enum MediaUploadError: Error {
    case invalidImageData
    case videoCompressionFailed
    case limitExceeded(kind: MediaUploadKind, originalBytes: Int, maxBytes: Int)
}

public struct PreparedUpload {
    public enum Body: Equatable {
        case data(Data)
        case file(URL)
    }

    public let kind: MediaUploadKind
    public let body: Body
    public let contentType: String
    public let originalBytes: Int
    public let finalBytes: Int
    public let context: String?

    public init(
        kind: MediaUploadKind,
        body: Body,
        contentType: String,
        originalBytes: Int,
        finalBytes: Int,
        context: String?
    ) {
        self.kind = kind
        self.body = body
        self.contentType = contentType
        self.originalBytes = originalBytes
        self.finalBytes = finalBytes
        self.context = context
    }
}

public actor MediaUploadPreprocessor {
    public static let shared = MediaUploadPreprocessor()

    private let maxImageBytes: Int
    private let maxVideoBytes: Int

    public init(
        maxImageBytes: Int = 10 * 1024 * 1024,
        maxVideoBytes: Int = 50 * 1024 * 1024
    ) {
        self.maxImageBytes = maxImageBytes
        self.maxVideoBytes = maxVideoBytes
    }

    public static func localizedLimitMessage(for kind: MediaUploadKind) -> String {
        let key = kind == .image ? "mediaUpload.toast.imageTooLarge" : "mediaUpload.toast.videoTooLarge"
        return NSLocalizedString(
            key,
            tableName: "Localizable",
            bundle: .module,
            value: "",
            comment: ""
        )
    }

    public func prepareImage(
        _ image: UIImage,
        contentType: String,
        uploadURL: String? = nil,
        context: String? = nil
    ) async throws -> PreparedUpload {
        let rawData: Data
        if contentType == "image/png" {
            guard let pngData = image.pngData() else {
                throw MediaUploadError.invalidImageData
            }
            rawData = pngData
        } else {
            guard let jpegData = image.jpegData(compressionQuality: 1) else {
                throw MediaUploadError.invalidImageData
            }
            rawData = jpegData
        }

        let originalBytes = rawData.count
        if originalBytes <= maxImageBytes {
            let prepared = PreparedUpload(
                kind: .image,
                body: .data(rawData),
                contentType: normalizedImageContentType(contentType),
                originalBytes: originalBytes,
                finalBytes: originalBytes,
                context: context
            )
            log(prepared, uploadURL: uploadURL)
            return prepared
        }

        var low = 0.05
        var high = 1.0
        var bestData = rawData
        for _ in 0..<10 {
            let quality = (low + high) / 2
            guard let data = image.jpegData(compressionQuality: quality) else {
                throw MediaUploadError.invalidImageData
            }
            if data.count <= maxImageBytes {
                bestData = data
                low = quality
            } else {
                high = quality
            }
        }

        if bestData.count > maxImageBytes {
            throw MediaUploadError.limitExceeded(
                kind: .image,
                originalBytes: originalBytes,
                maxBytes: maxImageBytes
            )
        }

        let prepared = PreparedUpload(
            kind: .image,
            body: .data(bestData),
            contentType: "image/jpeg",
            originalBytes: originalBytes,
            finalBytes: bestData.count,
            context: context
        )
        log(prepared, uploadURL: uploadURL)
        return prepared
    }

    public func prepareVideo(
        from sourceURL: URL,
        uploadURL: String? = nil,
        context: String? = nil
    ) async throws -> PreparedUpload {
        let originalBytes = try fileSizeBytes(at: sourceURL)
        if originalBytes <= maxVideoBytes {
            let prepared = PreparedUpload(
                kind: .video,
                body: .file(sourceURL),
                contentType: "video/mp4",
                originalBytes: originalBytes,
                finalBytes: originalBytes,
                context: context
            )
            log(prepared, uploadURL: uploadURL)
            return prepared
        }

        let asset = AVURLAsset(url: sourceURL)
        let presets = [
            AVAssetExportPreset1920x1080,
            AVAssetExportPreset1280x720,
            AVAssetExportPreset960x540,
            AVAssetExportPreset640x480,
            AVAssetExportPresetMediumQuality,
            AVAssetExportPresetLowQuality,
            "AVAssetExportPreset1280x720WithAppleProRes422",
            "AVAssetExportPreset1920x1080WithAppleProRes422"
        ]
        let compatiblePresets = AVAssetExportSession.exportPresets(compatibleWith: asset)

        for preset in presets where compatiblePresets.contains(preset) {
            let outputURL = FileManager.default.temporaryDirectory
                .appendingPathComponent("\(UUID().uuidString).mp4")
            do {
                let url = try await transcode(asset: asset, preset: preset, outputURL: outputURL)
                let finalBytes = try fileSizeBytes(at: url)
                if finalBytes <= maxVideoBytes {
                    let prepared = PreparedUpload(
                        kind: .video,
                        body: .file(url),
                        contentType: "video/mp4",
                        originalBytes: originalBytes,
                        finalBytes: finalBytes,
                        context: context
                    )
                    log(prepared, uploadURL: uploadURL)
                    return prepared
                }
            } catch {
                _ = try? FileManager.default.removeItem(at: outputURL)
            }
            _ = try? FileManager.default.removeItem(at: outputURL)
        }

        throw MediaUploadError.limitExceeded(
            kind: .video,
            originalBytes: originalBytes,
            maxBytes: maxVideoBytes
        )
    }

    private func transcode(
        asset: AVAsset,
        preset: String,
        outputURL: URL
    ) async throws -> URL {
        guard let exporter = AVAssetExportSession(asset: asset, presetName: preset) else {
            throw MediaUploadError.videoCompressionFailed
        }
        try? FileManager.default.removeItem(at: outputURL)
        exporter.outputURL = outputURL
        exporter.outputFileType = .mp4
        exporter.shouldOptimizeForNetworkUse = true

        try await withCheckedThrowingContinuation { continuation in
            exporter.exportAsynchronously {
                if exporter.status == .completed {
                    continuation.resume()
                } else {
                    continuation.resume(throwing: exporter.error ?? MediaUploadError.videoCompressionFailed)
                }
            }
        }
        return outputURL
    }

    private func normalizedImageContentType(_ contentType: String) -> String {
        contentType == "image/png" ? "image/png" : "image/jpeg"
    }

    private func fileSizeBytes(at url: URL) throws -> Int {
        guard let attributes = try? FileManager.default.attributesOfItem(atPath: url.path),
              let bytes = attributes[.size] as? NSNumber else {
            throw MediaUploadError.videoCompressionFailed
        }
        return bytes.intValue
    }

    public func log(_ prepared: PreparedUpload, uploadURL: String?) {
        let compressionRatio: Double
        if prepared.originalBytes > 0 {
            compressionRatio = 1 - Double(prepared.finalBytes) / Double(prepared.originalBytes)
        } else {
            compressionRatio = 0
        }

        let uploadPath: String
        if let uploadURL, let parsed = URL(string: uploadURL) {
            uploadPath = "\(parsed.host ?? "unknown")\(parsed.path)"
        } else {
            uploadPath = uploadURL ?? "local"
        }

        #if DEBUG
        let context = prepared.context ?? "default"
        print(
            "[MediaUploadPreprocessor] context=\(context) kind=\(prepared.kind.rawValue) " +
            "uploadURL=\(uploadPath) contentType=\(prepared.contentType) " +
            "originalBytes=\(prepared.originalBytes) finalBytes=\(prepared.finalBytes) " +
            "compressionRatio=\(String(format: "%.2f", compressionRatio))"
        )
        #endif
    }
}
