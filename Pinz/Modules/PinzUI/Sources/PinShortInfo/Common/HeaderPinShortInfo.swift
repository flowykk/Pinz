//
//  HeaderPinShortInfo.swift
//  PinzUI
//
//  Created by Danila Rakhmanov on 08.02.2026.
//

import SwiftUI
import PinzDomain

struct HeaderPinShortInfo: View {
    let pin: Pin
    let selectable: Bool
    let isSelected: Bool
    let onSelect: (() -> Void)?
    
    init(
        pin: Pin,
        selectable: Bool = false,
        isSelected: Bool = false,
        onSelect: (() -> Void)? = nil
    ) {
        self.pin = pin
        self.selectable = selectable
        self.isSelected = isSelected
        self.onSelect = onSelect
    }
    
    var body: some View {
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
                }.roundedFount(size: 34, weight: .light)
            }

            VStack(alignment: .leading) {
                HStack(spacing: 4) {
                    Image(systemName: "location.fill")
                    Text(pin.name)
                }.roundedFount(size: 16)

                Text(pin.category.value)
                    .roundedFount(size: 14, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
            }

            Spacer()

            VStack {
                Spacer(minLength: 0)
                HStack(spacing: 10) {
                    StatisticView(icon: "photo.stack.fill", text: String(pin.medias.count))
                    if !selectable {
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
