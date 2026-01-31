import SwiftUI
import PinzDomain

public struct PinAnnotationView: View {
    let pin: Pin

    @State private var currentMediaIndex = 0
    @State private var randomInterval: Double = 0

    var currentImage: UIImage? {
        guard !pin.medias.isEmpty,
              currentMediaIndex >= 0,
              currentMediaIndex < pin.medias.count else { return nil }
        guard case .image(let image) = pin.medias[currentMediaIndex].content else { return nil }
        return image
    }

    public init(pin: Pin) {
        self.pin = pin
        self.currentMediaIndex = currentMediaIndex
    }

    public var body: some View {
        VStack(spacing: 0) {
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

                    if pin.medias.count > 1 {
                        Text("\(pin.medias.count)")
                            .roundedFount(size: 12, weight: .semibold, foregroundColor: .black)
                            .frame(minWidth: 24, minHeight: 24)
                            .background(Circle().fill(Color.white))
                            .offset(x: 4, y: -4)
                    }
                }
            } else {
                EmptyView()
            }

            Triangle()
                .fill(Color.white)
                .frame(width: 24, height: 8)
                .offset(y: -1)
        }
        .compositingGroup()
        .shadow(color: .black.opacity(0.5), radius: 6, x: 0, y: 2)
        .onAppear {
            startImageRotation()
        }
    }

    private func startImageRotation() {
        randomInterval = Double.random(in: 7.0...10.0)
        guard pin.medias.count > 1 else { return }
        
        if currentMediaIndex >= pin.medias.count {
            currentMediaIndex = 0
        }

        Task {
            while true {
                try? await Task.sleep(nanoseconds: UInt64(randomInterval * 1_000_000_000))

                await MainActor.run {
                    guard !pin.medias.isEmpty else { return }
                    withAnimation(.easeInOut(duration: 1.3)) {
                        currentMediaIndex = (currentMediaIndex + 1) % pin.medias.count
                    }
                }
            }
        }
    }
}

// Вспомогательная фигура для треугольника-указателя
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
