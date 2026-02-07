import SwiftUI

extension UIImage {
    var isLight: Bool {
        guard let cgImage = self.cgImage else { return false }

        let width = cgImage.width
        let height = cgImage.height

        let sampleHeight = min(height / 3, 80)
        let sampleWidth = min(width / 3, 80)

        let rect = CGRect(
            x: 0,
            y: 0,
            width: sampleWidth,
            height: sampleHeight
        )

        guard let croppedCGImage = cgImage.cropping(to: rect) else {
            return false
        }

        let colorSpace = CGColorSpaceCreateDeviceRGB()
        let bitmapInfo = CGImageAlphaInfo.premultipliedLast.rawValue

        guard let context = CGContext(
            data: nil,
            width: Int(sampleWidth),
            height: Int(sampleHeight),
            bitsPerComponent: 8,
            bytesPerRow: Int(sampleWidth) * 4,
            space: colorSpace,
            bitmapInfo: bitmapInfo
        ) else {
            return false
        }

        context.draw(croppedCGImage, in: CGRect(x: 0, y: 0, width: sampleWidth, height: sampleHeight))

        guard let pixelData = context.data else { return false }

        let data = pixelData.bindMemory(to: UInt8.self, capacity: Int(sampleWidth * sampleHeight * 4))

        var totalBrightness: CGFloat = 0
        var pixelCount = 0

        for y in 0..<Int(sampleHeight) {
            for x in 0..<Int(sampleWidth) {
                let offset = (y * Int(sampleWidth) + x) * 4

                let r = CGFloat(data[offset]) / 255.0
                let g = CGFloat(data[offset + 1]) / 255.0
                let b = CGFloat(data[offset + 2]) / 255.0

                // Вычисляем яркость
                let brightness = 0.299 * r + 0.587 * g + 0.114 * b
                totalBrightness += brightness
                pixelCount += 1
            }
        }

        let averageBrightness = totalBrightness / CGFloat(pixelCount)
        return averageBrightness > 0.5
    }
}
