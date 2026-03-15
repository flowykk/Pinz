import SwiftUI

public struct CollapsibleImageView: View {

    let image: UIImage
    let cornetRadius: CGFloat

    public init(
        image: UIImage,
        cornetRadius: CGFloat = 24
    ) {
        self.image = image
        self.cornetRadius = cornetRadius
    }

    public var body: some View {
        CollapsibleView {
            Image(uiImage: image)
                .resizable()
                .aspectRatio(contentMode: .fill)
        }
        .clipped()
        .cornerRadius(cornetRadius)
    }
}
