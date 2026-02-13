import SwiftUI
import MapKit
import PinzUI
import PinzDomain

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
                        PinzButton(type: .slot(style: .secondary(needBorder: true), title: "Сбросить")) {
                            viewModel.dispatch(.reset)
                        }
                    }

                    PinzButton(type: .slot(style: .primary, title: "Готово")) {
                        viewModel.dispatch(.save)
                    }
                }
                .padding(.horizontal, 12)
                .background {
                    GradientView(style: .bottom, color: PinzUIAsset.background.swiftUIColor, opacity: 0.5, height: 120)
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
                    PinzButton(type: .icon(.chevronLeft), tint: .white) {
                        viewModel.dispatch(.back)
                    }
                },
                centerView: {
                    Text("Перемещай карту, чтобы\n изменить местоположение пина")
                        .multilineTextAlignment(.center)
                        .roundedFount(size: 14, foregroundColor: PinzUIAsset.background.swiftUIColor)
                }
            ).background {
                GradientView(style: .top, color: .black, opacity: 0.8, height: 200)
            }
            Spacer()
        }
    }
}
