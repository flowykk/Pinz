import SwiftUI
import PinzNavigation
import PinzTrips

import PinzBase
import PinzDomain
import PinzUI
import PinzPins

@main
struct PinzApp: App {
    @State private var router = AppRouter()

    let description = "В Сочи мы выбрались прошлым летом. Решили туда съездить, так как никогда раньше в этом городе- курорте не были, а это все же довольно популярное место. Многие в восторге от Сочи, город дорогой, популярный и престижный. Часто туда переезжают жить, ну или планируют это сделать в ближайшее время. Словом, город так и манит всех в свои теплые объятия, прямо как Москва. Ну вот и мы собрались на летний отдых туда, чтобы посмотреть, правда ли там так здорово. Для поездки выбрали месяц август. Да, это самый пик тур сезона, но вот так получилось. В Сочи ехали из Нижнего на поезде. На самом деле удобно, что без пересадок и до самого вокзала. Правда ехать пару суток, но я люблю поезда, поэтому даже в плацкарте на этот раз добрались без особых проблем. В поезде был вагон ресторан, и мы несколько раз заказывали еду из него."

    let pin = Pin(
        name: "Казань",
        description: "В Сочи мы выбрались прошлым летом. Решили туда съездить, так как никогда раньше в этом городе- курорте не были, а это все же довольно популярное место. Многие в восторге от Сочи, город дорогой, популярный и престижный. Часто туда переезжают жить, ну или планируют это сделать в ближайшее время. Словом, город так и манит всех в свои теплые объятия, прямо как Москва. Ну вот и мы собрались на летний отдых туда, чтобы посмотреть, правда ли там так здорово. Для поездки выбрали месяц август. Да, это самый пик тур сезона, но вот так получилось. В Сочи ехали из Нижнего на поезде. На самом деле удобно, что без пересадок и до самого вокзала. Правда ехать пару суток, но я люблю поезда, поэтому даже в плацкарте на этот раз добрались без особых проблем. В поезде был вагон ресторан, и мы несколько раз заказывали еду из него.",
        category: .entertainment,
        medias: [
            LoadedMedia(content: .image(PinzUIAsset.media1.image)),
            LoadedMedia(content: .image(PinzUIAsset.media10.image)),
            LoadedMedia(content: .image(PinzUIAsset.media2.image)),
            LoadedMedia(content: .image(PinzUIAsset.media3.image)),
            LoadedMedia(content: .image(PinzUIAsset.media4.image)),
            LoadedMedia(content: .image(PinzUIAsset.media9.image)),
            LoadedMedia(content: .image(PinzUIAsset.media5.image)),
            LoadedMedia(content: .image(PinzUIAsset.media6.image)),
            LoadedMedia(content: .image(PinzUIAsset.media11.image)),
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
//                PinInfoView(pin: pin)
                TripView(trip: Trip(
                    name: "Нижний Новгород",
                    image: PinzUIAsset.media3.image,
                    description: description,
                    pins: [pin, pin, pin, pin, pin],
                    season: .summer,
                    startDate: Date(fromDateString: "03.01.2026"),
                    endDate: Date(fromDateString: "09.01.2026"),
                    category: .vacation
                ))
            }.toolbar(.hidden)
        }
    }
}
