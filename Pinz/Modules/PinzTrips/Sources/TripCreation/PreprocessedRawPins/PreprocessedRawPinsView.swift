import SwiftUI
import PinzUI
import PinzDomain
import PinzPins

public struct PreprocessedRawPinsView: View {

    @State private var viewModel: PreprocessedRawPinsViewModel
    @State private var isMergePickerPresented = false

    @Environment(\.appRouter) private var router

    public init() {
        viewModel = PreprocessedRawPinsViewModel(
            pins: RawPins(
                pins: [
                    RawPin(medias: [
                        RawPinMedia(url: "https://i.pinimg.com/1200x/93/5d/50/935d504922bd5fd9597c5941dbb6c9ae.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/736x/ca/53/74/ca537401033425dc8dc8689884930b07.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/736x/eb/bc/27/ebbc278b59bbca831ee507f04020240d.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/736x/40/1d/4a/401d4a36dd09206dbb41d9969ff44dc2.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/1200x/90/17/a8/9017a826dedc6708ec0d825d9a222b1e.jpg", type: .image),
                    ]),
                    RawPin(medias: [
                        RawPinMedia(url: "https://i.pinimg.com/736x/34/cb/93/34cb93114fb0cca8f020cb9c26928394.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/736x/cb/f7/9b/cbf79b6388c70e03982a519436942256.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/1200x/c8/e5/d7/c8e5d7c87bdbc811b02c82344be63ad8.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/736x/75/28/1f/75281f11e4dc38b10d880d06cdd32cda.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/736x/e3/22/4f/e3224f8561b8eea36722c6b9c52788d3.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/1200x/7a/48/64/7a4864840c1fd55fd2f6613a66af9929.jpg", type: .image),
                    ]),
                    RawPin(medias: [
                        RawPinMedia(url: "https://i.pinimg.com/736x/59/79/59/5979594c0f0de1b583f60ce9ac15b94e.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/736x/dd/08/b4/dd08b40cee0b754035414222dd27ddc1.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/736x/29/9e/ff/299effcb075e97c1b4dc5ebcb7aac061.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/736x/1f/2d/c7/1f2dc7ba98b1c5c737e8942aab90751d.jpg", type: .image),
                        RawPinMedia(url: "https://i.pinimg.com/1200x/14/e3/88/14e388399238e64b67bed42e0541c8d9.jpg", type: .image),
                    ])
                ]
            )
        )
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                header
            } content: {
                if !viewModel.isLoading {
                    content
                }
            }

            if viewModel.isLoading {
                LoadingView()
            } else {
                gradientWithButtons
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
        .mergePinsSheet(isPresented: $isMergePickerPresented, pins: viewModel.pins.pins) { first, second in
            viewModel.dispatch(.mergePins(firstIndex: first, secondIndex: second))
        }
    }

    @ViewBuilder
    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { viewModel.dispatch(.navigate(.back)) }
            )
        })
    }

    private var content: some View {
        VStack {
            let pins = viewModel.pins.pins
            ForEach(pins.indices, id: \.self) { index in
                RawPinView(
                    pin: pins[index],
                    index: index,
                    allPins: pins,
                    onDeleteMedia: { media in
                        viewModel.dispatch(.deleteMedia(media, fromPin: pins[index].id))
                    },
                    onMoveMedia: { media, targetIndex in
                        viewModel.dispatch(.moveMedia(media, fromPin: index, toPin: targetIndex))
                    }
                )
                .padding(.horizontal, 12)
                if index != pins.count - 1 {
                    Divider().padding(.leading, 12)
                }
            }
        }.padding(.bottom, 170)
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons(height: 190) {
            VStack(spacing: 6) {
                HStack(spacing: 6) {
                    PinzButton(
                        type: .slot(style: .secondary(needBorder: true), title: "Объединить пины"),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        disabled: viewModel.pins.pins.count < 2,
                        action: .plain { isMergePickerPresented = true }
                    )

                    PinzButton(
                        type: .slot(style: .secondary(needBorder: true), title: "Добавить пин"),
                        tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                        action: .plain { viewModel.dispatch(.addPin) }
                    )
                }
            }

            PinzButton(
                type: .slot(style: .primary, title: "Далее"),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                disabled: false,
                action: .async { try await viewModel.asyncDispatch(.continue) }
            )
        }
    }
}
