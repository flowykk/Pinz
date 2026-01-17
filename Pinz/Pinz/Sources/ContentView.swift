import SwiftUI
import PinzAuthentication
import PinzProfile
import PinzTrips
import PinzDomain
import PinzUI

public struct ContentView: View {
    public init() {}

    public var body: some View {
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

//        AuthFlowView()
//        SettingsView()
//        ProfileView()
        TripView(
            trip: Trip(
                name: "Нижний Новгород",
                image: PinzUIAsset.avatar.image,
                pins: [pin, pin, pin, pin, pin]
            )
        )
    }
}

struct ContentView_Previews: PreviewProvider {
    static var previews: some View {
        ContentView()
    }
}
