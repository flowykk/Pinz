import SwiftUI
import PinzNavigation
import PinzTrips

import PinzDomain
import PinzUI

@main
struct PinzApp: App {
    @State private var router = AppRouter()

    let pin = Pin(
        name: "Казань",
        category: "Природа",
        medias: [
            LoadedMedia(content: .image(PinzUIAsset.media1.image)),
            LoadedMedia(content: .image(PinzUIAsset.media2.image)),
            LoadedMedia(content: .image(PinzUIAsset.media3.image)),
            LoadedMedia(content: .image(PinzUIAsset.media4.image)),
            LoadedMedia(content: .image(PinzUIAsset.media5.image)),
            LoadedMedia(content: .image(PinzUIAsset.media6.image)),
            LoadedMedia(content: .image(PinzUIAsset.media7.image)),
            LoadedMedia(content: .image(PinzUIAsset.media8.image)),
        ],
        privacy: true,
        tags: [
            MediaTag(tag: "Машины"),
            MediaTag(tag: "Достопримечательность"),
            MediaTag(tag: "ГАЗ"),
            MediaTag(tag: "Экскурсия"),
        ]
    )

    var body: some Scene {
        WindowGroup {
            RootView(router: router) {
                TripView(trip: Trip(
                    name: "Нижний Новгород",
                    image: PinzUIAsset.avatar.image,
                    pins: [pin, pin, pin, pin, pin]
                ))
            }
        }
    }
}
