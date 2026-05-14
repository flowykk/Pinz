import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

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
                    PinzButton(
                        type: .icon(.chevronLeft),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { viewModel.dispatch(.navigate(.back())) }
                    )
                }, centerView: {
                    HeaderTitle(PinzBaseStrings.PostPreview.Title.main)
                })
            } content: {
                VStack(spacing: 16) {
                    Group {
                        tripMap
                        desription
                    }.padding(.horizontal, 12)

                    pinsList
                        .padding(.bottom, 90)
                }
            }

            BottomGradientWithButtons {
                PinzButton(
                    type: .slot(style: .primary, title: PinzBaseStrings.PostPreview.Button.publish),
                    tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                    disabled: viewModel.isPublishing,
                    action: .async {
                        await viewModel.asyncDispatch(.publish) { error in
                            viewModel.publishError = error.localizedDescription
                        }
                    }
                )
            }
        }
        .onAppear { viewModel.setRouter(router) }
        .alert(PinzBaseStrings.PostPreview.Error.publishFailed, isPresented: Binding(
            get: { viewModel.publishError != nil },
            set: { isPresented in
                if !isPresented { viewModel.publishError = nil }
            }
        )) {
            Button(PinzBaseStrings.Common.Button.ok) {
                viewModel.publishError = nil
            }
        } message: {
            Text(viewModel.publishError ?? "")
        }
        .overlay(alignment: .center) {
            if viewModel.isPublishing {
                LoadingView()
            }
        }
    }

    public var tripMap: some View {
        TripMapView(position: $viewModel.position, pins: viewModel.selectedPins)
            .aspectRatio(1, contentMode: .fit)
            .clipShape(RoundedRectangle(cornerRadius: 26))
            .disabled(true)
    }

    public var desription: some View {
        DescriptionView(description: viewModel.trip.description)
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
