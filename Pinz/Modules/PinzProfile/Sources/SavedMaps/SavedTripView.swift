import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

public struct SavedTripView: View {

    @State private var viewModel: SavedTripViewModel
    @Environment(\.appRouter) private var router

    public init(trip: Trip) {
        viewModel = SavedTripViewModel(trip: trip)
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                Header(leftView: {
                    PinzButton(
                        type: .icon(.chevronLeft),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { viewModel.dispatch(.navigate(.back)) }
                    )
                }, centerView: {
                    HeaderTitle(viewModel.trip.name)
                }, rightView: {
                    Button {
                        Task { await viewModel.toggleSaved() }
                    } label: {
                        Image(systemName: viewModel.isSaved ? "bookmark.fill" : "bookmark")
                            .foregroundStyle(PinzUIAsset.textPrimary.swiftUIColor)
                            .frame(minWidth: 44, minHeight: 44)
                    }
                    .buttonStyle(.plain)
                    .disabled(viewModel.isTogglingSaved)
                })
            } content: {
                VStack(spacing: 16) {
                    Group {
                        tripMap
                        DescriptionView(description: viewModel.trip.description)
                    }
                    .padding(.horizontal, 12)

                    DefaultPinsListView(
                        pins: viewModel.pins,
                        hideMediaBadges: true,
                        pinTapped: { _ in }
                    )
                    .padding(.bottom, 20)
                }
                .padding(.top, 8)
            }
            .background(PinzUIAsset.background.swiftUIColor)

            if viewModel.isLoading {
                LoadingView()
            }
        }
        .onAppear {
            viewModel.setRouter(router)
            Task { await viewModel.loadTrip() }
        }
        .alert(PinzBaseStrings.SavedTrip.Error.load, isPresented: Binding(
            get: { viewModel.loadError != nil },
            set: { isPresented in
                if !isPresented { viewModel.loadError = nil }
            }
        )) {
            Button(PinzBaseStrings.Common.Button.ok) {
                viewModel.loadError = nil
            }
        } message: {
            Text(viewModel.loadError ?? "")
        }
    }

    private var tripMap: some View {
        TripMapView(position: $viewModel.position, pins: viewModel.pins)
            .aspectRatio(1, contentMode: .fit)
            .disabled(true)
            .overlay {
                VStack {
                    Spacer()
                    GradientView(style: .bottom, color: .black, height: 120)
                }
                .padding(.bottom, -85)
            }
            .clipShape(RoundedRectangle(cornerRadius: 26))
    }
}
