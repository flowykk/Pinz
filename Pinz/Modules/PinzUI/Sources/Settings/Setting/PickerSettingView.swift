import SwiftUI
import PinzDomain

//extension Setting.PickerSetting {
//    @ViewBuilder
//    var view: some View {
//        Button {
//            // TODO: show picker
//        } label: {
//            settingView
//        }
//    }
//
//    private var settingView: some View {
//        HStack(spacing: 0) {
//            if let icon {
//                Image(systemName: icon.rawValue)
//                    .roundedFount(size: 18, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
//                    .frame(16)
//                    .padding(.trailing, 12)
//            }
//            Text(title)
//                .roundedFount(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
//
//            Spacer()
//            
//            if let currentValue = value.wrappedValue {
//                switch currentValue.content {
//                case .text(let text):
//                    Text(text)
//                        .roundedFount(size: 16, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
//                }
//            }
//
//            Image(systemName: "chevron.right")
//                .roundedFount(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
//                .padding(.leading, 8)
//        }.frame(height: 52)
//    }
//}
