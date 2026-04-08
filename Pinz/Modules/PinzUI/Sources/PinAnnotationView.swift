import SwiftUI
import PinzDomain
import PinzBase

public struct PinAnnotationView: View {
    let pin: Pin

    @State private var currentMediaIndex = 0
    @State private var randomInterval: Double = 0
    @State private var loadedImages: [Int: UIImage] = [:]
    @State private var isLoadingImage = false

    var currentImage: UIImage? {
        guard !pin.medias.isEmpty,
              currentMediaIndex >= 0,
              currentMediaIndex < pin.medias.count else { return nil }
        return loadedImages[currentMediaIndex]
    }

    public init(pin: Pin) {
        self.pin = pin
    }

    public var body: some View {
        VStack(spacing: 0) {
            Group {
                if let image = currentImage {
                    ZStack(alignment: .topTrailing) {
                        ZStack {
                            Image(uiImage: image)
                                .resizable()
                                .scaledToFill()
                                .frame(62)
                                .clipShape(RoundedRectangle(cornerRadius: 22))
                                .id(currentMediaIndex)

                            RoundedRectangle(cornerRadius: 18)
                                .strokeBorder(Color.white, lineWidth: 4)
                                .frame(62)
                        }
                    }
                } else {
                    ZStack {
                        RoundedRectangle(cornerRadius: 22)
                            .fill(Color.gray.opacity(0.3))
                            .frame(62)

                        if isLoadingImage {
                            ProgressView()
                                .tint(.white)
                        }

                        RoundedRectangle(cornerRadius: 18)
                            .strokeBorder(Color.white, lineWidth: 4)
                            .frame(62)
                    }
                }
            }.overlay {
                if pin.medias.count > 1 {
                    Text("\(pin.medias.count)")
                        .roundedFont(size: 12, weight: .semibold, foregroundColor: .black)
                        .frame(minWidth: 24, minHeight: 24)
                        .background(Circle().fill(Color.white))
                        .offset(x: 25, y: -25)
                }
            }

            Triangle()
                .fill(Color.white)
                .frame(width: 24, height: 8)
                .offset(y: -1)
        }
        .compositingGroup()
        .shadow(color: .black.opacity(0.5), radius: 6, x: 0, y: 2)
        .task {
            await preloadAllImages()
            startImageRotation()
        }
    }

    private func preloadAllImages() async {
        guard !pin.medias.isEmpty else { return }
        
        isLoadingImage = true
        
        // Загружаем все изображения параллельно в фоне
        await withTaskGroup(of: (Int, UIImage?).self) { group in
            for (index, media) in pin.medias.enumerated() {
                guard let url = media.mediaURL else { continue }
                
                group.addTask {
                    let image = await ImageProvider.loadOrGetImage(for: url.absoluteString, .media)
                    return (index, image)
                }
            }
            
            for await (index, image) in group {
                await MainActor.run {
                    if let image = image {
                        loadedImages[index] = image
                        // Устанавливаем currentMediaIndex на первое загруженное изображение
                        if loadedImages.count == 1 {
                            currentMediaIndex = index
                        }
                    }
                    if index == currentMediaIndex {
                        isLoadingImage = false
                    }
                }
            }
        }
    }

    private func startImageRotation() {
        guard pin.medias.count > 1 else { return }
        
        if currentMediaIndex >= pin.medias.count {
            currentMediaIndex = 0
        }

        Task {
            randomInterval = generateRandomInterval()
            
            while true {
                try? await Task.sleep(nanoseconds: UInt64(randomInterval * 1_000_000_000))

                await MainActor.run {
                    guard pin.medias.count > 1 else { return }
                    
                    // Получаем индексы успешно загруженных изображений
                    let loadedIndices = loadedImages.keys.sorted()
                    guard !loadedIndices.isEmpty else { return }
                    
                    // Если загружено больше одного изображения, выбираем случайное
                    if loadedIndices.count > 1 {
                        var newIndex = loadedIndices.randomElement() ?? currentMediaIndex
                        while newIndex == currentMediaIndex && loadedIndices.count > 1 {
                            newIndex = loadedIndices.randomElement() ?? currentMediaIndex
                        }
                        
                        withAnimation(.easeInOut(duration: 1.3)) {
                            currentMediaIndex = newIndex
                        }
                    }
                    
                    randomInterval = generateRandomInterval()
                }
            }
        }
    }

    private func generateRandomInterval() -> Double {
        Double.random(in: 7.0...10.0)
    }
}

struct Triangle: Shape {
    func path(in rect: CGRect) -> Path {
        var path = Path()
        path.move(to: CGPoint(x: rect.midX, y: rect.maxY))
        path.addLine(to: CGPoint(x: rect.minX, y: rect.minY))
        path.addLine(to: CGPoint(x: rect.maxX, y: rect.minY))
        path.closeSubpath()
        return path
    }
}
