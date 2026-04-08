import SwiftUI
import MapKit
import PinzUI
import PinzDomain
import PinzBase

public struct PinPlaceChangeView: View {

    @State private var viewModel: PinPlaceChangeViewModel

    @Environment(\.appRouter) private var router
    @AppStorage("pinzMapStyle") private var mapStyleRawValue: String = PinzMapStyle.satelight.rawValue

    public init(pin: Pin, onSave: @escaping (CLLocationCoordinate2D) -> Void) {
        self._viewModel = State(initialValue: PinPlaceChangeViewModel(pin: pin, onSave: onSave))
    }

    public var body: some View {
        ZStack {
            Map(position: $viewModel.position)
                .savedMapStyle(mapStyleRawValue)
                .mapControlVisibility(.hidden)
                .ignoresSafeArea()
                .onMapCameraChange { context in
                    viewModel.dispatch(.update(context))
                }

            header

            VStack {
                Spacer()
                PinAnnotationView(pin: viewModel.pin)
                    .offset(y: -48)
                    .allowsHitTesting(false)
                Spacer()
            }

            VStack {
                Spacer()

                HStack(spacing: 12) {
                    if viewModel.hasChanges {
                        PinzButton(
                            type: .slot(style: .secondary(needBorder: true), title: PinzBaseStrings.Common.Button.reset),
                            action: .plain { viewModel.dispatch(.reset) }
                        )
                    }

                    PinzButton(
                        type: .slot(style: .primary, title: PinzBaseStrings.Common.Button.done),
                        action: .plain { viewModel.dispatch(.save) }
                    )
                }
                .padding(.horizontal, 12)
                .background {
                    GradientView(style: .bottom, color: PinzUIAsset.background.swiftUIColor, opacity: 0.3, height: 100)
                        .padding(.bottom, -30)
                }
                .animation(.easeOut(duration: 0.3), value: viewModel.hasChanges)
            }
        }
        .onAppear { viewModel.setRouter(router) }
    }

    private var header: some View {
        VStack(spacing: 0) {
            Header(
                backgroundColor: .clear,
                leftView: {
                    PinzButton(
                        type: .icon(.chevronLeft),
                        tint: .white,
                        action: .plain { viewModel.dispatch(.back) }
                    )
                },
                centerView: {
                    Text(PinzBaseStrings.PinPlaceChange.Label.instructions)
                        .multilineTextAlignment(.center)
                        .roundedFont(size: 14, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                }
            ).background {
                GradientView(style: .top, color: .black, height: 200)
                    .padding(.top, -50)
            }
            Spacer()
        }
    }
}
