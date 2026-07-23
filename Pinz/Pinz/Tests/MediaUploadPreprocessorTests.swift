import XCTest
@testable import PinzBase
import UIKit
import CoreGraphics

final class MediaUploadPreprocessorTests: XCTestCase {

    func test_prepareImage_keepsImageUnderLimitWithoutCompression() async throws {
        let preprocessor = MediaUploadPreprocessor(maxImageBytes: 1_500_000)
        let image = makeNoisyImage(size: CGSize(width: 64, height: 64))

        let prepared = try await preprocessor.prepareImage(
            image,
            contentType: "image/jpeg",
            context: "test"
        )

        let originalBytes = image.jpegData(compressionQuality: 1)?.count ?? 0
        XCTAssertLessThanOrEqual(prepared.originalBytes, prepared.finalBytes)
        XCTAssertEqual(prepared.finalBytes, prepared.originalBytes)
        XCTAssertEqual(prepared.originalBytes, originalBytes)
        XCTAssertEqual(prepared.kind, .image)
        XCTAssertEqual(prepared.body, .data(image.jpegData(compressionQuality: 1) ?? Data()))
    }

    func test_prepareImage_compressesImageWhenOverLimit() async throws {
        // Noisy 1024² JPEG often stays above very small caps even at minimum quality.
        let maxBytes = 1_200_000
        let preprocessor = MediaUploadPreprocessor(maxImageBytes: maxBytes)
        let image = makeNoisyImage(size: CGSize(width: 1024, height: 1024))
        let originalBytes = image.jpegData(compressionQuality: 1)?.count ?? 0

        XCTAssertGreaterThan(originalBytes, maxBytes)

        let prepared = try await preprocessor.prepareImage(image, contentType: "image/jpeg", context: "test")

        XCTAssertLessThanOrEqual(prepared.finalBytes, maxBytes)
        XCTAssertLessThan(prepared.finalBytes, prepared.originalBytes)
        XCTAssertEqual(prepared.kind, .image)
    }

    func test_prepareImage_throwsLimitExceededWhenCompressionCannotFit() async {
        let preprocessor = MediaUploadPreprocessor(maxImageBytes: 1)
        let image = makeNoisyImage(size: CGSize(width: 64, height: 64))

        do {
            _ = try await preprocessor.prepareImage(image, contentType: "image/jpeg", context: "test")
            XCTFail("Expected image compression to fail")
        } catch let error as MediaUploadError {
            if case .limitExceeded = error {
                XCTAssertTrue(true)
            } else {
                XCTFail("Expected limitExceeded, got \(error)")
            }
        } catch {
            XCTFail("Expected MediaUploadError, got \(error)")
        }
    }

    func test_prepareVideo_keepsFileUnderLimit() async throws {
        let preprocessor = MediaUploadPreprocessor(maxVideoBytes: 1_000_000)
        let mediaURL = makeTempFile(name: "small-video.mp4", bytes: Data([1, 2, 3, 4, 5, 6]))
        defer { try? FileManager.default.removeItem(at: mediaURL) }

        let prepared = try await preprocessor.prepareVideo(from: mediaURL, context: "test")

        XCTAssertEqual(prepared.kind, .video)
        XCTAssertEqual(prepared.finalBytes, prepared.originalBytes)
        switch prepared.body {
        case .file(let preparedURL):
            XCTAssertEqual(preparedURL, mediaURL)
        case .data:
            XCTFail("Expected file body for untouched video")
        }
    }

    func test_prepareVideo_throwsLimitExceededForUncompressedSource() async {
        let preprocessor = MediaUploadPreprocessor(maxVideoBytes: 1)
        let mediaURL = makeTempFile(name: "large-video.mp4", bytes: Data(repeating: 0, count: 2_048))
        defer { try? FileManager.default.removeItem(at: mediaURL) }

        do {
            _ = try await preprocessor.prepareVideo(from: mediaURL, context: "test")
            XCTFail("Expected video compression to fail")
        } catch let error as MediaUploadError {
            if case .limitExceeded = error {
                XCTAssertTrue(true)
            } else {
                XCTFail("Expected limitExceeded, got \(error)")
            }
        } catch {
            XCTFail("Expected MediaUploadError, got \(error)")
        }
    }

    private func makeTempFile(name: String, bytes: Data) -> URL {
        let fileURL = FileManager.default.temporaryDirectory
            .appendingPathComponent(name)
        try? bytes.write(to: fileURL, options: .atomic)
        return fileURL
    }

    private func makeNoisyImage(size: CGSize) -> UIImage {
        let width = Int(size.width)
        let height = Int(size.height)
        let bytesPerPixel = 4
        let bytesPerRow = width * bytesPerPixel
        var pixels = [UInt8](repeating: 0, count: width * height * bytesPerPixel)

        for y in 0..<height {
            for x in 0..<width {
                let index = (y * width + x) * bytesPerPixel
                pixels[index] = UInt8((x * 97 + y * 193) & 0xFF)
                pixels[index + 1] = UInt8((x * 43 + y * 29) & 0xFF)
                pixels[index + 2] = UInt8((x * 73 + y * 17) & 0xFF)
                pixels[index + 3] = 255
            }
        }

        let provider = CGDataProvider(data: Data(pixels) as CFData)
        let colorSpace = CGColorSpaceCreateDeviceRGB()
        let bitmapInfo = CGBitmapInfo.byteOrder32Big.rawValue | CGImageAlphaInfo.premultipliedLast.rawValue
        let cgImage = CGImage(
            width: width,
            height: height,
            bitsPerComponent: 8,
            bitsPerPixel: 32,
            bytesPerRow: bytesPerRow,
            space: colorSpace,
            bitmapInfo: CGBitmapInfo(rawValue: bitmapInfo),
            provider: provider!,
            decode: nil,
            shouldInterpolate: false,
            intent: .defaultIntent
        )!

        return UIImage(cgImage: cgImage)
    }
}
