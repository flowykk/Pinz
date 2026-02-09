import SwiftUI
import PinzUI
import PinzDomain

public struct PostPreviewView: View {

    @State private var viewModel: PostPreviewViewModel

    @Environment(\.appRouter) private var router
    @Environment(\.dismiss) private var dismiss

    public init(
        trip: Trip,
        selectedPins: [Pin],
    ) {
        viewModel = PostPreviewViewModel(trip: trip, selectedPins: selectedPins)
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                Header(leftView: {
                    PinzButton(type: .icon(.chevronLeft), tint: PinzUIAsset.textPrimary.swiftUIColor) {
                        viewModel.dispatch(.navigate(.back()))
                    }
                }, centerView: {
                    HeaderTitle("Так будет выглядеть пост")
                })
            } content: {
                VStack(spacing: 16) {
                    tripMap
                        .padding(.horizontal, 12)

                    pinsList
                        .padding(.bottom, 90)
                }
            }

            BottomGradientWithButtons {
                PinzButton(
                    type: .slot(style: .primary, title: "Опубликовать путешествие"),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor
                ) {
                    router?.pop(by: 2)
                }
            }
        }
        .onAppear { viewModel.setRouter(router) }
    }

    public var tripMap: some View {
        TripMapView(position: $viewModel.position, pins: viewModel.selectedPins)
            .aspectRatio(1, contentMode: .fit)
            .clipShape(RoundedRectangle(cornerRadius: 26))
            .disabled(true)
    }

    public var pinsList: some View {
        DefaultPinsListView(
            pins: viewModel.selectedPins,
            hideMediaBadges: true,
            pinTapped: { pin in
                viewModel.dispatch(.navigate(.pinInfo(pin)))
            },
        )
    }
}
