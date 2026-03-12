import SwiftUI

public struct MediaBadgesView<
    LeadingTopBadge: View,
    LeadingBottomBadge: View,
    TrailingTopBadge: View,
    TrailingBottomBadge: View,
>: View {

    @ViewBuilder private let leadingTopBadge: LeadingTopBadge
    @ViewBuilder private let leadingBottomBadge: LeadingBottomBadge
    @ViewBuilder private let trailingTopBadge: TrailingTopBadge
    @ViewBuilder private let trailingBottomBadge: TrailingBottomBadge

    public init(
        @ViewBuilder leadingTopBadge: () -> LeadingTopBadge = { EmptyView() },
        @ViewBuilder leadingBottomBadge: () -> LeadingBottomBadge = { EmptyView() },
        @ViewBuilder trailingTopBadge: () -> TrailingTopBadge = { EmptyView() },
        @ViewBuilder trailingBottomBadge: () -> TrailingBottomBadge = { EmptyView() },
    ) {
        self.leadingTopBadge = leadingTopBadge()
        self.leadingBottomBadge = leadingBottomBadge()
        self.trailingTopBadge = trailingTopBadge()
        self.trailingBottomBadge = trailingBottomBadge()
    }


    public var body: some View {
        HStack(spacing: 0) {
            VStack(spacing: 0) {
                leadingTopBadge
                Spacer()
                leadingBottomBadge
            }
            Spacer()
            VStack(spacing: 0) {
                trailingTopBadge
                Spacer()
                trailingBottomBadge
            }
        }
    }
}
