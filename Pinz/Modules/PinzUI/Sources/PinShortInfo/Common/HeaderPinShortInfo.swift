//
//  HeaderPinShortInfo.swift
//  PinzUI
//
//  Created by Danila Rakhmanov on 08.02.2026.
//

import SwiftUI
import PinzDomain

public struct HeaderPinShortInfo: View {
    let pin: Pin
    let selectable: Bool
    let isSelected: Bool
    let isPrivacyShown: Bool
    let onSelect: (() -> Void)?
    
    public init(
        pin: Pin,
        selectable: Bool = false,
        isSelected: Bool = false,
        isPrivacyShown: Bool = true,
        onSelect: (() -> Void)? = nil
    ) {
        self.pin = pin
        self.selectable = selectable
        self.isSelected = isSelected
        self.isPrivacyShown = isPrivacyShown
        self.onSelect = onSelect
    }
    
    public var body: some View {
        HStack {
            if selectable {
                VStack {
                    Spacer(minLength: 0)
                    if pin.isPrivate {
                        Image(systemName: "lock.fill").foregroundStyle(PinzUIAsset.accentRed.swiftUIColor)
                    } else {
                        Button {
                            onSelect?()
                        } label: {
                            Image(systemName: isSelected ? "checkmark.square.fill" : "square")
                        }
                        .buttonStyle(.plain)
                        .transaction { $0.animation = .easeInOut(duration: 0.15) }
                    }
                    Spacer(minLength: 0)
                }.roundedFont(size: 34, weight: .light)
            }

            VStack(alignment: .leading) {
                HStack(spacing: 4) {
                    Image(systemName: "location.fill")
                    Text(pin.name)
                }.roundedFont(size: 16)

                Text(pin.category.value)
                    .roundedFont(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
            }

            Spacer()

            VStack {
                Spacer(minLength: 0)
                HStack(spacing: 10) {
                    StatisticView(icon: "photo.stack.fill", text: String(pin.medias.count))
                    if !selectable && isPrivacyShown {
                        StatisticView(
                            icon: pin.isPrivate == true ? "lock.fill" : "lock.open.fill",
                            iconColor: pin.isPrivate == true ? PinzUIAsset.accentRed : PinzUIAsset.accentGreen
                        )
                    }
                }
                Spacer(minLength: 0)
            }
        }.padding(.horizontal, 16)
    }
}
