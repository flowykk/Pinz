import Foundation
import CoreLocation

public struct Pin: Hashable, Identifiable {
    public var id: String { name }
    
    public var name: String
    public var description: String?
    public var category: PinCategory
    public var medias: [MediaItem]
    public var isPrivate: Bool
    public var startDate: Date?
    public var endDate: Date?
    public var tags: [MediaTag]
    public var coordinates: CLLocationCoordinate2D

    public init(
        name: String,
        description: String? = nil,
        category: PinCategory,
        medias: [MediaItem],
        isPrivate: Bool,
        startDate: Date? = nil,
        endDate: Date? = nil,
        tags: [MediaTag],
        coordinates: CLLocationCoordinate2D
    ) {
        self.name = name
        self.description = description
        self.category = category
        self.medias = medias
        self.isPrivate = isPrivate
        self.startDate = startDate
        self.endDate = endDate
        self.tags = tags
        self.coordinates = coordinates
    }
}

extension CLLocationCoordinate2D: Hashable {
    public func hash(into hasher: inout Hasher) {
        hasher.combine(latitude)
        hasher.combine(longitude)
    }
    
    public static func == (lhs: CLLocationCoordinate2D, rhs: CLLocationCoordinate2D) -> Bool {
        lhs.latitude == rhs.latitude && lhs.longitude == rhs.longitude
    }
}

extension Pin {
    public static func stubs() -> [Pin] {
        [
            Pin(
                name: "Храм Христа Спасителя",
                description: "Кафедральный собор Русской православной церкви. Находится на левом берегу Москвы-реки, недалеко от Кремля. Построен в память о победе над Наполеоном. Храм является одним из самых высоких православных храмов в мире.",
                category: .entertainment,
                medias: [MediaItem(
                    isPrivate: Bool.random(),
                    type: .video,
                    mediaURL: URL(string: "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg"),
//                    mediaURL: URL(string: "https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/720/Big_Buck_Bunny_720_10s_1MB.mp4"),
                )] + urlStringsToMediaItems(for: [
                    "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg",
                    "https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg",
                    "https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg",
                    "https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg",
                    "https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg",
                    "https://i.pinimg.com/736x/34/cb/93/34cb93114fb0cca8f020cb9c26928394.jpg",
                    "https://i.pinimg.com/736x/cb/f7/9b/cbf79b6388c70e03982a519436942256.jpg",
                    "https://i.pinimg.com/1200x/c8/e5/d7/c8e5d7c87bdbc811b02c82344be63ad8.jpg",
                    "https://i.pinimg.com/736x/75/28/1f/75281f11e4dc38b10d880d06cdd32cda.jpg",
                    "https://i.pinimg.com/736x/e3/22/4f/e3224f8561b8eea36722c6b9c52788d3.jpg",
                    "https://i.pinimg.com/1200x/7a/48/64/7a4864840c1fd55fd2f6613a66af9929.jpg",
                    "https://i.pinimg.com/736x/59/79/59/5979594c0f0de1b583f60ce9ac15b94e.jpg",
                    "https://i.pinimg.com/736x/dd/08/b4/dd08b40cee0b754035414222dd27ddc1.jpg",
                    "https://i.pinimg.com/736x/29/9e/ff/299effcb075e97c1b4dc5ebcb7aac061.jpg",
                    "https://i.pinimg.com/736x/1f/2d/c7/1f2dc7ba98b1c5c737e8942aab90751d.jpg",
                    "https://i.pinimg.com/1200x/14/e3/88/14e388399238e64b67bed42e0541c8d9.jpg",
                    "https://i.pinimg.com/736x/9f/9e/3e/9f9e3ebec1472e02b3563c7c1e15970c.jpg",
                    "https://i.pinimg.com/1200x/a2/aa/69/a2aa69b55a2d0119bf477c1984bb5c77.jpg",
                    "https://avatars.mds.yandex.net/i?id=d19b5e25223f5583c129bc11983be385_l-13234616-images-thumbs&n=13",
                ]),
                isPrivate: Bool.random(),
                tags: [
                    MediaTag(tag: "Религия"),
                    MediaTag(tag: "Достопримечательность"),
                    MediaTag(tag: "Архитектура"),
                    MediaTag(tag: "История")
                ],
                coordinates: CLLocationCoordinate2D(
                    latitude: 55.7447,
                    longitude: 37.6055
                )
            ),
            Pin(
                name: "Красная площадь",
                description: "Главная площадь России",
                category: .entertainment,
                medias: urlStringsToMediaItems(for: [
                    "https://i.pinimg.com/736x/38/31/13/383113a2561e6a4c92973a30027bb70c.jpg",
                    "https://i.pinimg.com/736x/12/5b/a4/125ba4fee18a2921a42ab76ac02bf578.jpg",
                    "https://i.pinimg.com/1200x/39/4a/f3/394af3064a6edc4abca76d0820fcd725.jpg",
                    "https://i.pinimg.com/736x/70/13/e5/7013e510c6ca3a000d15989fcf12e5f0.jpg",
                    "https://i.pinimg.com/736x/8a/ba/48/8aba48fff9da9a3af7f57348411f01b0.jpg",
                    "https://i.pinimg.com/1200x/83/3d/4e/833d4ec2c8b7afe0593de70d09823443.jpg",
                    "https://i.pinimg.com/736x/f5/ce/ef/f5ceef7cf315cb31474d66a41e093b13.jpg",
                    "https://i.pinimg.com/736x/ae/8a/82/ae8a82bf58576132b123e8cb2f1b15f9.jpg",
                    "https://i.pinimg.com/736x/38/43/63/384363cca8065d1a00a7be6ca00d9a96.jpg",
                    "https://i.pinimg.com/736x/37/6f/7c/376f7c4ee53425178116ff0737645e6c.jpg",
                    "https://i.pinimg.com/1200x/59/4d/c9/594dc91e3c9115c6639531f54e28f484.jpg",
                    "https://i.pinimg.com/736x/b1/d4/07/b1d4074af9450d9ce0b6f2fe5db8f36c.jpg",
                    "https://i.pinimg.com/736x/06/dc/fa/06dcfa6e1a3aaf1539724b3d48f21280.jpg",
                    "https://i.pinimg.com/736x/2f/0b/16/2f0b16ad2c349d732a53b97ae30932f2.jpg",
                ]),
                isPrivate: Bool.random(),
                tags: [
                    MediaTag(tag: "Достопримечательность")
                ],
                coordinates: CLLocationCoordinate2D(
                    latitude: 55.7539,
                    longitude: 37.6208
                )
            ),
            Pin(
                name: "Парк Горького",
                description: "Центральный парк культуры и отдыха",
                category: .nature,
                medias: urlStringsToMediaItems(for: [
                    "https://i.pinimg.com/1200x/cd/47/23/cd4723e7bac0a34506e84b9c378d9eaf.jpg",
                    "https://i.pinimg.com/1200x/a9/e8/67/a9e867ac241af016ee06bea2cd5b5abb.jpg",
                    "https://i.pinimg.com/1200x/66/a0/94/66a094638921cfd9e7a3ce009bc43409.jpg",
                    "https://i.pinimg.com/1200x/5f/2e/15/5f2e1561dc3ddd63cb50435e360a6abb.jpg",
                    "https://i.pinimg.com/736x/20/37/67/2037675effb87b929f6978c94eede4a0.jpg",
                    "https://i.pinimg.com/736x/26/7e/cb/267ecb23d231f486c433678a3db964a7.jpg",
                    "https://i.pinimg.com/1200x/58/21/ff/5821ff51e338a076a747194e780d124a.jpg",
                    "https://i.pinimg.com/736x/56/27/89/56278965ca1b43924ae7d5c21ae37764.jpg",
                    "https://i.pinimg.com/736x/1c/f0/2f/1cf02f94d8800d6a172c3f4e554eb512.jpg",
                    "https://i.pinimg.com/736x/45/c8/68/45c868423bd19cda2e472eff7a61e0dd.jpg",
                    "https://i.pinimg.com/736x/ce/ed/3a/ceed3ae1cc1c839743f3b2cb4a4a2c23.jpg",
                    "https://i.pinimg.com/1200x/e2/b4/26/e2b426206dbb0b1cc832c80e2d9259ee.jpg",
                ]),
                isPrivate: Bool.random(),
                tags: [
                    MediaTag(tag: "Парк")
                ],
                coordinates: CLLocationCoordinate2D(
                    latitude: 55.7312,
                    longitude: 37.6014
                )
            ),
            Pin(
                name: "Москва-Сити",
                description: "Деловой центр Москвы",
                category: .entertainment,
                medias: [MediaItem(
                    isPrivate: Bool.random(),
                    type: .video,
                    mediaURL: URL(string: "https://test-videos.co.uk/vids/bigbuckbunny/mp4/h264/720/Big_Buck_Bunny_720_10s_1MB.mp4"),
                )] + urlStringsToMediaItems(for: [
                    "https://i.pinimg.com/736x/5b/b1/e3/5bb1e361d85340e6e7051dbf4d2c45ee.jpg",
                    "https://i.pinimg.com/736x/ad/71/99/ad7199d4ba597372294020f617061c0c.jpg",
                    "https://i.pinimg.com/736x/c1/f2/8c/c1f28c92e5cf4f992ccffba31206f006.jpg",
                    "https://i.pinimg.com/736x/b6/c4/08/b6c408540892d66a0f54b905afe0ab54.jpg",
                    "https://i.pinimg.com/736x/0c/a7/d3/0ca7d3ff242a9a7cad6176659da03017.jpg",
                    "https://i.pinimg.com/736x/b5/9f/58/b59f58fce89f9abe810b24cd7d9c8820.jpg",
                ]),
                isPrivate: Bool.random(),
                tags: [
                    MediaTag(tag: "Архитектура")
                ],
                coordinates: CLLocationCoordinate2D(
                    latitude: 55.7496,
                    longitude: 37.5369
                )
            ),
            Pin(
                name: "Воробьевы горы",
                description: "Смотровая площадка с видом на Москву",
                category: .nature,
                medias: urlStringsToMediaItems(for: [
                         "https://i.pinimg.com/736x/77/65/ac/7765ac5175540792659b036142c9a49d.jp",
                         "https://i.pinimg.com/736x/09/ed/11/09ed11941e1ed576ced2d9614ac72486.jpg",
                         "https://i.pinimg.com/736x/4a/6d/5d/4a6d5d7b63c7ea87f284a170cb72800d.jpg",
                         "https://i.pinimg.com/1200x/4c/2e/97/4c2e979c9db604b92b8c5e512e31e48b.jpg",
                         "https://i.pinimg.com/736x/ba/92/a6/ba92a63f7cd3fe42d67c22675d43faf9.jpg",
                         "https://i.pinimg.com/736x/94/d8/f1/94d8f1a38c0c029c524ad6e7c2c6cefd.jpg",
                         "https://i.pinimg.com/736x/b9/34/ea/b934ead5ea6f8c051807a85489a0daeb.jpg",
                         "https://i.pinimg.com/736x/c2/8b/49/c28b493f5a7ae67163e70705690ab28a.jpg",
                         "https://i.pinimg.com/736x/31/ab/c2/31abc28ac9d79ac90099a6ca8c6dbbd3.jpg",
                         "https://i.pinimg.com/736x/7f/67/50/7f67509f179dffb5682d65081464ed71.jpg",
                         "https://avatars.mds.yandex.net/i?id=4df40aaf6f820181ce16396e0d10bfc9_l-5474553-images-thumbs&n=13",
                         "https://avatars.mds.yandex.net/get-altay/14271118/2a00000192de44e4f2ce2fcde5de16b48d80/orig",
                         "https://ucare.timepad.ru/8d3e632c-9906-4ea5-962f-67682b51f89c/-/preview/",
                         "https://www.mos.ru/upload/newsfeed/newsfeed/tramplinGL(9).jp",
                ]),
                isPrivate: Bool.random(),
                tags: [
                    MediaTag(tag: "Природа")
                ],
                coordinates: CLLocationCoordinate2D(
                    latitude: 55.7105,
                    longitude: 37.5425
                )
            )
        ]
    }

    private static func urlStringsToMediaItems(for urls: [String]) -> [MediaItem] {
        urls.enumerated().map { index, urlString in
            MediaItem(
                id: index + 1,
                isPrivate: Bool.random(),
                type: .image,
                mediaURL: URL(string: urlString)
            )
        }
    }
}
