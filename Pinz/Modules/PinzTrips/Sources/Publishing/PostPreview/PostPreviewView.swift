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
                    TripMapView(position: $viewModel.position, pins: viewModel.selectedPins)
                        .padding(.horizontal, 12)
                        .aspectRatio(1, contentMode: .fit)
                        .clipShape(RoundedRectangle(cornerRadius: 26))
                        .disabled(true)

                    VStack {
                        ForEach(viewModel.selectedPins.indices, id: \.self) { index in
                            DefaultPinShortInfoView(
                                pin: viewModel.selectedPins[index],
                                hideMediaBadges: true,
                                pinTapped: { pin in
                                    viewModel.dispatch(.navigate(.pinInfo(pin)))
                                },
                            )
                            if index != viewModel.selectedPins.count - 1 {
                                Divider().padding(.leading, 12)
                            }
                        }
                    }
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
}
