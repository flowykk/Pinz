import SwiftUI

public struct WishlistElement {
    public let image: UIImage
    public let title: String
    public let description: String

    public init(image: UIImage, title: String, description: String) {
        self.image = image
        self.title = title
        self.description = description
    }
}

public extension WishlistElement {
    public static var stubs: [WishlistElement] {
        [
            WishlistElement(
                image: PinzDomainAsset.machu.image,
                title: "Мачу-Пикчу",
                description: "Мечтаю посетить древний город в горах Перу, увидеть руины инков на рассвете и почувствовать дух истории."
            ),
            WishlistElement(
                image: PinzDomainAsset.kyoto.image,
                title: "Киото",
                description: "Хочу прогуляться по японским садам, посмотреть цветение сакуры у храмов и попробовать настоящую японскую кухню."
            ),
            WishlistElement(
                image: PinzDomainAsset.iceland.image,
                title: "Исландия",
                description: "Мечтаю увидеть северное сияние, поплавать в Голубой лагуне, прокатиться по кольцевой дороге и полюбоваться водопадами."
            ),
            WishlistElement(
                image: PinzDomainAsset.newYork.image,
                title: "Нью-Йорк",
                description: "Очень хочу увидеть Таймс-сквер, подняться на небоскрёб, сходить в Центральный парк и прокатиться на метро."
            ),
        ]
    }
}
