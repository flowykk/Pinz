import SwiftUI
import PinzDomain
import PinzUI

struct PinAnnotationView: View {
    let pin: Pin
    
    var body: some View {
        VStack(spacing: 0) {
            if let firstMedia = pin.medias.first,
               case .image(let image) = firstMedia.content {
                Image(uiImage: image)
                    .resizable()
                    .scaledToFill()
                    .frame(width: 56, height: 56)
                    .clipShape(RoundedRectangle(cornerRadius: 18))
                    .overlay(
                        RoundedRectangle(cornerRadius: 18)
                            .strokeBorder(Color.white, lineWidth: 4)
                    )
            } else {
                EmptyView()
            }
            
            Triangle()
                .fill(Color.white)
                .frame(width: 24, height: 8)
                .offset(y: -1)
        }.shadow(color: .black.opacity(0.3), radius: 4, x: 0, y: 2)
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
