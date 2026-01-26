import SwiftUI
import PinzDomain

public struct TagsView: View {
    var tags: [MediaTag]
    @State private var totalHeight: CGFloat
    var onTagAdd: ((MediaTag) -> Void)?
    var onTagDelete: ((MediaTag) -> Void)?

    public init(
        tags: [MediaTag],
        totalHeight: Double = CGFloat.zero,
        onTagAdd: ((MediaTag) -> Void)?,
        onTagDelete: ((MediaTag) -> Void)?
    ) {
        self.tags = tags
        self.totalHeight = totalHeight
        self.onTagAdd = onTagAdd
        self.onTagDelete = onTagDelete
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            GeometryReader { geometry in
                self.generateContent(in: geometry)
            }
            .frame(height: totalHeight)
        }
    }

    // swiftlint:disable function_body_length
    private func generateContent(in geometry: GeometryProxy) -> some View {
        var width: CGFloat = 0
        var height: CGFloat = 0

        var lastWidth: CGFloat = 0
        var lastHeight: CGFloat = 0

        return ZStack(alignment: .topLeading) {
            ForEach(self.tags) { tag in
                self.item(for: tag.tag)
                    .padding([.top, .trailing], 4)
                    .alignmentGuide(.leading) { dimension in
                        if abs(width - dimension.width) > geometry.size.width {
                            width = 0
                            height -= dimension.height
                        }
                        let result = width
                        if let lastTag = self.tags.last, tag == lastTag {
                            lastWidth = -abs(width - dimension.width)
                            width = 0
                        } else {
                            width -= dimension.width
                        }
                        return result
                    }
                    .alignmentGuide(.top) { _ in
                        let result = height
                        if let lastTag = self.tags.last, tag == lastTag {
                            lastHeight = height
                            height = 0
                        }
                        return result
                    }
            }

//            Image(systemName: "plus")
//                .font(.system(size: 18, weight: .black))
//                .frame(width: 38, height: 38)
//                .foregroundColor(PinzUIAsset.textPrimary.swiftUIColor)
//                .clipShape(Circle())
//                .padding([.vertical, .trailing], 4)
//                .alignmentGuide(.leading) { dimension in
//                    if abs(lastWidth - dimension.width) > geometry.size.width {
//                        lastWidth = 0
//                        lastHeight -= dimension.height
//                    }
//                    return lastWidth
//                }
//                .alignmentGuide(.top) { _ in return lastHeight }
//                .disabledWithOpacity(tags.count >= maxTags)
        }
        .onGeometryChange(for: CGSize.self) { proxy in
            return proxy.size
        } action: { newSize in
//            withAnimation {
                totalHeight = newSize.height
//            }
        }
    }
    // swiftlint:enable function_body_length

    private func item(for text: String) -> some View {
        HStack {
            Text(text)
                .foregroundColor(PinzUIAsset.textPrimary.swiftUIColor)
                .roundedFount(size: 14, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                .padding(.horizontal, 8)

//            Image(systemName: "minus")
//                .font(.system(size: 16, weight: .black))
//                .frame(width: 30, height: 30)
//                .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
//                .foregroundColor(PinzUIAsset.backgroundSecondary.swiftUIColor)
//                .clipShape(Circle())
//                .onTapGesture {
//                    withAnimation {
//                        onTagDelete?(MediaTag(tag: text))
//                    }
//                }
        }
        .padding(8)
        .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
        .cornerRadius(20)
    }
}
