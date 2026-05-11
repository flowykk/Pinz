import SwiftUI
import PinzUI
import PinzDomain
import PinzBase

public struct TripPinsListView: View {

    @State private var viewModel: TripPinsListViewModel

    @Environment(\.appRouter) private var router
    @Environment(\.showToast) private var showToast

    public init(trip: Trip) {
        viewModel = TripPinsListViewModel(trip: trip)
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
                    HeaderTitle(PinzBaseStrings.TripPins.title)
                })
            } content: {
                pinsList
            }

            if viewModel.trip.pins.isEmpty {
                NoPinsPlaceholderView()
            }

            gradientWithButtons
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear {
            viewModel.setRouter(router)
            viewModel.setShowToast(showToast)
            viewModel.refreshActiveSessionFlag()
        }
    }

    @ViewBuilder
    private var pinsList: some View {
        if viewModel.trip.pins.isEmpty {
            EmptyView()
        } else {
            DefaultPinsListView(
                pins: viewModel.trip.pins,
                pinTapped: { pin in
                    viewModel.dispatch(.navigate(.pinInfo(pin)))
                },
                onMediaUpdated: { updatedMedia, pin in
                    guard let pinIdx = viewModel.trip.pins.firstIndex(where: { $0.serverId == pin.serverId }),
                          let mediaIdx = viewModel.trip.pins[pinIdx].medias.firstIndex(where: { $0.mediaId == updatedMedia.mediaId }) else { return }
                    viewModel.trip.pins[pinIdx].medias[mediaIdx] = updatedMedia
                }
            ).padding(.bottom, 90)
        }
    }

    private var addMediaStatusLabel: String? {
        switch viewModel.trip.status {
        case "ADD_MEDIA_UPLOADING": return PinzBaseStrings.TripPins.Status.uploading
        case "ADD_MEDIA_GROUPING_REVIEW": return PinzBaseStrings.TripPins.Status.groupingReview
        case "ADD_MEDIA_PROCESSING": return PinzBaseStrings.TripPins.Status.processing
        case "ADD_MEDIA_DRAFT_FINAL_REVIEW": return PinzBaseStrings.TripPins.Status.draftFinalReview
        default: return nil
        }
    }

    private var pinUploadStatusLabel: String? {
        viewModel.hasActivePinUploadSession ? "Создание пина..." : nil
    }

    private var hasAnyStatusLabel: Bool {
        addMediaStatusLabel != nil || pinUploadStatusLabel != nil
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            HStack(alignment: .top, spacing: 6) {
                VStack(spacing: 4) {
                    PinzButton(
                        type: .slot(style: .secondary(needBorder: true), title: PinzBaseStrings.TripPins.Button.addMedia),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        action: .plain { Task { try? await viewModel.asyncDispatch(.addMedia) } }
                    )
                    addMediaStatusLabelView
                }

                VStack(spacing: 4) {
                    PinzButton(
                        type: .slot(style: .primary, title: PinzBaseStrings.TripPins.Button.addPin),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        action: .plain { Task { try? await viewModel.asyncDispatch(.addPin) } }
                    )
                    pinUploadStatusLabelView
                }
            }.if(hasAnyStatusLabel) { view in
                view.padding(.bottom, -20)
            }
        }
    }

    @ViewBuilder
    private var addMediaStatusLabelView: some View {
        if let label = addMediaStatusLabel {
            Text(label)
                .roundedFont(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                .multilineTextAlignment(.center)
        }
    }

    @ViewBuilder
    private var pinUploadStatusLabelView: some View {
        if let label = pinUploadStatusLabel {
            Text(label)
                .roundedFont(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                .multilineTextAlignment(.center)
        }
    }
}
