import SwiftUI
import PinzDomain

public struct MediaGridView: View {

    public struct Item: Identifiable, Hashable {
        public var id: String
        public var url: String
        public var type: MediaType

        public init(id: String, url: String, type: MediaType) {
            self.id = id
            self.url = url
            self.type = type
        }
    }

    private let items: [Item]
    private let cornerRadius: CGFloat
    private let columns = Array(repeating: GridItem(.flexible(), spacing: 4), count: 4)

    public init(items: [Item], cornerRadius: CGFloat = 14) {
        self.items = items
        self.cornerRadius = cornerRadius
    }

    public var body: some View {
        LazyVGrid(columns: columns, spacing: 4) {
            ForEach(items) { item in
                MediaThumbnailView(
                    url: URL(string: item.url),
                    type: item.type,
                    contentMode: .fill,
                    cornerRadius: cornerRadius,
                    square: true
                )
            }
        }
    }
}
