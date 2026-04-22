import SwiftUI
import PinzDomain
import PinzUI
import PinzBase

public struct RawPinView: View {

    private let pin: RawPin
    private let index: Int
    private let allPins: [RawPin]
    private let movablePinIds: Set<String>?
    private let onDeleteMedia: ((RawPinMedia) -> Void)?
    private let onMoveMedia: ((RawPinMedia, Int) -> Void)?
    private let isMediaLocked: (RawPinMedia) -> Bool

    private let columns = Array(repeating: GridItem(.flexible(), spacing: 4), count: 4)

    private var movablePins: [(globalIndex: Int, pin: RawPin)] {
        let allowed = movablePinIds
        return allPins.indices
            .filter { $0 != index }
            .filter { targetIndex in
                guard let movablePinIds else { return true }
                return movablePinIds.contains(allPins[targetIndex].id)
            }
            .map { (globalIndex: $0, pin: allPins[$0]) }
    }

    public init(
        pin: RawPin,
        index: Int,
        allPins: [RawPin] = [],
        onDeleteMedia: ((RawPinMedia) -> Void)? = nil,
        onMoveMedia: ((RawPinMedia, Int) -> Void)? = nil,
        isMediaLocked: @escaping (RawPinMedia) -> Bool = { _ in false },
        movablePinIds: Set<String>? = nil
    ) {
        self.pin = pin
        self.index = index
        self.allPins = allPins
        self.onDeleteMedia = onDeleteMedia
        self.onMoveMedia = onMoveMedia
        self.isMediaLocked = isMediaLocked
        self.movablePinIds = movablePinIds
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            header
            if !pin.medias.isEmpty {
                LazyVGrid(columns: columns, spacing: 4) {
                    ForEach(pin.medias) { media in
                        MediaThumbnailCell(
                            media: media,
                            movablePins: movablePins,
                            onDeleteMedia: onDeleteMedia,
                            onMoveMedia: onMoveMedia,
                            isLocked: isMediaLocked(media)
                        )
                    }
                }
            }
        }
    }

    private var header: some View {
        HStack {
            VStack(alignment: .leading) {
                HStack(spacing: 4) {
                    Image(systemName: "location.fill")
                    Text(PinzBaseStrings.Common.Label.pinNumber(index + 1))
                }.roundedFont(size: 16)
            }

            Spacer()

            VStack {
                Spacer(minLength: 0)
                HStack(spacing: 10) {
                    StatisticView(icon: "photo.stack.fill", text: String(pin.medias.count))
                }
                Spacer(minLength: 0)
            }
        }.padding(.horizontal, 4)
    }
}

private struct MediaThumbnailCell: View {

    let media: RawPinMedia
    let movablePins: [(globalIndex: Int, pin: RawPin)] 
    let onDeleteMedia: ((RawPinMedia) -> Void)?
    let onMoveMedia: ((RawPinMedia, Int) -> Void)?
    let isLocked: Bool

    @State private var isMovePickerPresented = false

    var body: some View {
        let isInteractive = !isLocked

        RawPinMediaThumbnailView(
            media: media,
            contentMode: .fill,
            cornerRadius: 14,
            square: true
        )
        .overlay {
            MediaBadgesView(
                leadingTopBadge: {
                    if media.type == .video && !isLocked {
                        BadgeView(icon: .video)
                    }
                },
                trailingTopBadge: {
                    if isLocked {
                        BadgeView(icon: .lock, color: PinzUIAsset.accentRed.swiftUIColor)
                    }
                }
            ).padding(4)
        }
        .disabledWithOpacity(isLocked, opacity: 0.4)
        .contextMenu {
            if isInteractive {
                if movablePins.count > 0 {
                    Button {
                        isMovePickerPresented = true
                    } label: {
                        Label(PinzBaseStrings.RawPin.Button.move, systemImage: "arrow.left.arrow.right")
                    }
                }
                Button(role: .destructive) { onDeleteMedia?(media) } label: {
                    Label(PinzBaseStrings.Common.Button.delete, systemImage: "trash")
                }
            }
        } preview: {
            RawPinMediaThumbnailView(
                media: media,
                contentMode: .fill,
                cornerRadius: 14
            )
        }
        .movePinMediaSheet(isPresented: $isMovePickerPresented, movablePins: movablePins) { globalIndex in
            if isInteractive {
                onMoveMedia?(media, globalIndex)
            }
        }
    }
}
