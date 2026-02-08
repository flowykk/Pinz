import SwiftUI
import PinzDomain

public struct TagsView: View {

    public enum Style {
        case `default`
        case editing
    }

    var tags: [MediaTag]
    @State private var totalHeight: CGFloat
    let onTagAdd: ((MediaTag) -> Void)?
    let onTagDelete: ((MediaTag) -> Void)?
    let style: Style

    @State private var isAddTagPresented: Bool = false
    @State private var height: CGFloat = 0

    public init(
        tags: [MediaTag],
        totalHeight: Double = CGFloat.zero,
        onTagAdd: ((MediaTag) -> Void)?,
        onTagDelete: ((MediaTag) -> Void)?,
        style: Style,
    ) {
        self.tags = tags
        self.totalHeight = totalHeight
        self.onTagAdd = onTagAdd
        self.onTagDelete = onTagDelete
        self.style = style
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

            if style == .editing {
                Button {
                    isAddTagPresented = true
                } label: {
                    Image(systemName: "plus")
                        .frame(32)
                        .roundedFount(size: 14, weight: .bold, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                        .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
                        .clipShape(Circle())
                        .padding([.vertical, .trailing], 4)
                        .alignmentGuide(.leading) { dimension in
                            if abs(lastWidth - dimension.width) > geometry.size.width {
                                lastWidth = 0
                                lastHeight -= dimension.height
                            }
                            return lastWidth
                        }
                        .alignmentGuide(.top) { _ in return lastHeight }
//                        .disabledWithOpacity(tags.count >= maxTags)
                }.buttonStyle(.plain)
            }
        }
        .onGeometryChange(for: CGSize.self) { proxy in
            return proxy.size
        } action: { newSize in
            withAnimation {
                totalHeight = newSize.height
            }
        }
        .sheet(isPresented: $isAddTagPresented) {
            AddTagView(onTagAdd: onTagAdd)
                .pinzSheet()
                .presentationDetents([.height(120)])
                .background(PinzUIAsset.background.swiftUIColor)
        }
    }
    // swiftlint:enable function_body_length

    private func item(for text: String) -> some View {
        HStack(spacing: 0) {
            Text(text)
                .roundedFount(size: 14, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                .padding(.leading, 8)
                .padding(.trailing, style == .editing ? 4 : 8)

            if style == .editing {
                Button {
                    withAnimation {
                        onTagDelete?(MediaTag(tag: text))
                    }
                } label: {
                    Image(systemName: "minus")
                        .frame(24)
                        .roundedFount(size: 12, weight: .bold, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                        .background(PinzUIAsset.background.swiftUIColor)
                        .clipShape(Circle())
                }
            }
        }
        .padding(4)
        .frame(height: 32)
        .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
        .cornerRadius(20)
    }
}
