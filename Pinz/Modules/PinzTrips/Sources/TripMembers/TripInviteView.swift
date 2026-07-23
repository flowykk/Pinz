import CoreImage
import PinzBase
import PinzUI
import SwiftUI
import UIKit

struct TripInviteView: View {

    @State private var viewModel: TripInviteViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var isSharePresented = false

    init(tripId: String) {
        _viewModel = State(initialValue: TripInviteViewModel(tripId: tripId))
    }

    var body: some View {
        VStack(spacing: 0) {
            Header(
                leftView: {
                    PinzButton(
                        type: .icon(.xmark),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { dismiss() }
                    )
                },
                centerView: {
                    HeaderTitle(PinzBaseStrings.TripMembers.Title.invite)
                }
            )

            Group {
                if viewModel.isLoading {
                    Spacer()
                    LoadingView()
                    Spacer()
                } else if let error = viewModel.errorMessage {
                    inviteErrorView(message: error)
                } else if let urlString = viewModel.inviteUrl {
                    inviteContent(urlString: urlString)
                } else {
                    Spacer()
                    LoadingView()
                    Spacer()
                }
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .sheet(isPresented: $isSharePresented) {
            if let url = URL(string: viewModel.inviteUrl ?? "") {
                ShareActivityView(items: [url])
            }
        }
        .task {
            await viewModel.load()
        }
    }

    @ViewBuilder
    private func inviteErrorView(message: String) -> some View {
        VStack(spacing: 16) {
            Spacer()
            Text(message)
                .roundedFont(size: 16, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 24)
            PinzButton(
                type: .text(PinzBaseStrings.Common.Button.retry),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .async { await viewModel.load() }
            )
            Spacer()
        }
    }

    @ViewBuilder
    private func inviteContent(urlString: String) -> some View {
        ScrollView {
            VStack(spacing: 24) {
                if let qr = TripInviteQRCoder.uiImage(from: urlString) {
                    Image(uiImage: qr)
                        .interpolation(.none)
                        .resizable()
                        .scaledToFit()
                        .frame(maxWidth: 220, maxHeight: 220)
                        .padding(.top, 24)
                }

                VStack(spacing: 12) {
                    PinzButton(
                        type: .slot(style: .primary, title: PinzBaseStrings.TripMembers.Button.shareLink),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { isSharePresented = true }
                    )
                    .disabledWithOpacity(URL(string: urlString) == nil)

                    PinzButton(
                        type: .slot(style: .secondary(needBorder: false), title: PinzBaseStrings.TripMembers.Button.copyLink),
                        tint: PinzUIAsset.textPrimary.swiftUIColor,
                        action: .plain { copyToPasteboard(urlString) }
                    )
                }
                .padding(.horizontal, 24)
                .padding(.bottom, 32)
            }
        }
    }

    private func copyToPasteboard(_ string: String) {
        UIPasteboard.general.string = string
    }
}

private enum TripInviteQRCoder {
    static func uiImage(from string: String) -> UIImage? {
        let data = Data(string.utf8)
        guard let filter = CIFilter(name: "CIQRCodeGenerator") else { return nil }
        filter.setValue(data, forKey: "inputMessage")
        filter.setValue("H", forKey: "inputCorrectionLevel")
        guard let output = filter.outputImage else { return nil }
        let scale: CGFloat = 12
        let scaled = output.transformed(by: CGAffineTransform(scaleX: scale, y: scale))
        let context = CIContext()
        guard let cgImage = context.createCGImage(scaled, from: scaled.extent) else { return nil }
        return UIImage(cgImage: cgImage)
    }
}
