import SwiftUI
import MapKit
import PinzUI
import PinzDomain
import PinzBase

struct PostView: View {

    @State private var viewModel: PostViewModel
    @State private var selection: Int = 0

    init(post: Post) {
        viewModel = PostViewModel(post: post)
    }

    var body: some View {
        VStack(spacing: 0) {
            TabView(selection: $selection.animation()) {
                map
                pins
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
            .frame(height: 300)
            .padding(.top, 4)

            TabViewProgressView(numberOfPages: viewModel.post.pins.count + 1, currentIndex: selection)
                .padding(.top, 12)

            statistics
                .padding(.top, 8)
                .padding(.horizontal, 12)
        }
        .task {
            await viewModel.loadImages()
        }
    }

    public var statistics: some View {
        HStack(spacing: 10) {
            StatisticView(
                icon: "hand.thumbsup",
                text: String(viewModel.post.likes),
                iconSize: 16
            )
            StatisticView(
                icon: "hand.thumbsdown",
                iconSize: 16
            )
            StatisticView(
                icon: "bookmark",
                iconSize: 16
            )
            Spacer()
            StatisticView(
                icon: "eye",
                text: String(viewModel.post.views),
                iconSize: 16
            )
        }
    }

    public var map: some View {
        TripMapView(
            position: $viewModel.position,
            pins: viewModel.post.pins
        )
        .disabled(true)
        .overlay {
            VStack {
                GradientView(style: .top, color: PinzUIAsset.background.swiftUIColor, height: 150)
                Spacer()
            }.padding(.top, -60)
        }
        .overlay {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 0) {
                    Text(viewModel.post.name).roundedFont(size: 20, weight: .bold, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                    Text("Отдых, Лето").roundedFont(size: 14, weight: .semibold, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                    Spacer()
                }
                Spacer()
                StatisticView(
                    icon: "person.2.fill",
                    text: String(viewModel.post.participants),
                    iconSize: 16,
                    iconColor: PinzUIAsset.textPrimary
                )
            }
            .padding(.horizontal, 14)
            .padding(.top, 10)
        }
        .disabled(true)
        .cornerRadius(10)
        .tag(0)
    }

    public var pins: some View {
        ForEach(viewModel.post.pins.indices, id: \.self) { index in
            let pin = viewModel.post.pins[index]
            Group {
                if let image = viewModel.images[index] {
                    Image(uiImage: image)
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .clipped()
                } else {
                    Rectangle()
                        .fill(Color.gray.opacity(0.3))
                        .overlay { ProgressView().tint(.white) }
                }
            }
            .frame(width: UIScreen.main.bounds.width, height: 300)
            .clipped()
            .overlay {
                VStack {
                    Spacer()
                    GradientView(style: .bottom, color: .black, height: 200)
                }
                .padding(.bottom, -100)
            }
            .overlay {
                HStack {
                    VStack(alignment: .leading, spacing: 0) {
                        Spacer()
                        Text(pin.name).roundedFont(size: 16, weight: .bold, foregroundColor: .white)
                        Text(pin.category.value).roundedFont(size: 10, weight: .semibold, foregroundColor: .white)
                    }
                    Spacer()
                }
                .padding(.leading, 14)
                .padding(.bottom, 10)
            }
            .cornerRadius(10)
            .tag(index + 1)
        }
    }
}
