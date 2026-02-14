import SwiftUI
import PinzBase
import PinzDomain

struct PinStoryCardView: View {

    var pin: Pin
    let pins: [Pin]
    @Binding var currentStory: String
    @State var timer = Timer.publish(every: 0.05, on: .main, in: .common).autoconnect()
    @State var timerProgress: CGFloat = 0
    @State var cachedImages: [Int: UIImage] = [:]
    @State var isFirstMediaCached = false
    @State var isPaused = false

    @Environment(\.dismiss) private var dismiss

    var body: some View {
        let index = min(Int(timerProgress), pin.medias.count - 1)
        let media = pin.medias[index]
        let image = cachedImages[media.id]

        if index < pin.medias.count {
            GeometryReader { proxy in
                ZStack(alignment: .center) {
                    if let image {
                        storyBackground(for: image)
                            .overlay {
                                Image(uiImage: image)
                                    .resizable()
                                    .aspectRatio(contentMode: .fit)
                            }
                    } else {
                        Rectangle()
                            .fill(Color.gray.opacity(0.3))
                            .aspectRatio(1, contentMode: .fit)
                            .overlay {
                                ProgressView()
                                    .tint(.white)
                            }
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .overlay(alignment: .top) {
                    GradientView(style: .top, color: .black, opacity: 0.5, height: 130, needsBlur: false)
                }
                .overlay { tappableAreas }
                .overlay(alignment: .topTrailing) { interface }
                .overlay(alignment: .top) { progressiveCapsules }
                .onAppear {
                    timerProgress = 0
                    Task {
                        await precacheMedias()
                    }
                }
                .onReceive(timer) { _ in
                    guard currentStory == pin.id,
                          isFirstMediaCached,
                          !isPaused else { return }
                    
                    if timerProgress < CGFloat(pin.medias.count) {
                        timerProgress += 0.01
                    } else {
                        updateStory()
                    }
                }
                .rotation3DEffect(
                    getAngle(proxy: proxy),
                    axis: (x: 0, y: 1, z: 0),
                    anchor: proxy.frame(in: .global).minX > 0 ? .leading : .trailing
                )
                .ignoresSafeArea(edges: .bottom)
            }
        }
    }

    private var tappableAreas: some View {
        HStack {
            Rectangle()
                .fill(.black.opacity(0.01))
                .onTapGesture {
                    if (timerProgress - 1) < 0 {
                        updateStory(forward: false)
                    } else {
                        timerProgress = CGFloat(Int(timerProgress - 1))
                    }
                }

            Rectangle()
                .fill(.black.opacity(0.01))
                .onTapGesture {
                    if (timerProgress + 1) > CGFloat(pin.medias.count) {
                        updateStory()
                    } else {
                        timerProgress = CGFloat(Int(timerProgress + 1))
                    }
                }
        }.onLongPressGesture(minimumDuration: 0.1) {
            isPaused = true
        } onPressingChanged: { pressing in
            if !pressing {
                isPaused = false
            }
        }
    }

    private func storyBackground(for image: UIImage) -> some View {
        Image(uiImage: image)
            .resizable()
            .aspectRatio(contentMode: .fill)
            .frame(width: UIScreen.main.bounds.width)
            .clipped()
            .overlay(.ultraThinMaterial)
            .overlay(.black.opacity(0.4))
            .overlay(alignment: .bottom) {
                GradientView(style: .bottom, color: .black, opacity: 0.5, height: 130, needsBlur: false)
            }
    }

    private var interface: some View {
        HStack(spacing: 13) {
            Text(pin.name)
                .fontWeight(.bold)
                .foregroundColor(.white)

            Spacer()

            Button {
                dismiss()
            } label: {
                Rectangle().fill(.clear).frame(32)
                    .overlay(alignment: .trailing) {
                        Image(systemName: "xmark")
                            .roundedFount(size: 20, foregroundColor: .white)
                    }
            }
        }
        .padding()
        .padding(.top, 4)
    }

    private var progressiveCapsules: some View {
        HStack(spacing: 3) {
            ForEach(pin.medias.indices, id: \.self) { index in
                GeometryReader { proxy in
                    let width = proxy.size.width
                    let progress = timerProgress - CGFloat(index)
                    let perfectProgress = min(max(progress, 0), 1)

                    Capsule()
                        .fill(.gray.opacity(0.5))
                        .overlay(alignment: .leading) {
                            Capsule()
                                .fill(.white)
                                .frame(width: width * perfectProgress)
                        }
                }
            }
        }
        .frame(height: 1.5)
        .padding(.horizontal)
        .padding(.top, 8)
    }

    func updateStory(forward: Bool = true) {
        let index = min(Int(timerProgress), pin.medias.count - 1)
        let media = pin.medias[index]

        if !forward {
            if let first = pins.first, first.id != pin.id {
                let bundleIndex = pins.firstIndex { currentBundle in
                    return pin.id == currentBundle.id
                } ?? 0

                currentStory = pins[bundleIndex - 1].id
            } else {
                timerProgress = 0
            }
        }

        if let last = pin.medias.last, last.id == media.id {
            if let lastBundle = pins.last, lastBundle.id == pin.id {
                dismiss()
            } else {
                let bundleIndex = pins.firstIndex { currentBundle in
                    return pin.id == currentBundle.id
                } ?? 0

                currentStory = pins[bundleIndex + 1].id
            }
        }
    }

    func getAngle(proxy: GeometryProxy) -> Angle {
        let progress = proxy.frame(in: .global).minX / proxy.size.width

        let rotationAngle: CGFloat = 45
        let degrees = rotationAngle * progress

        return Angle(degrees: Double(degrees))
    }

    private func precacheMedias() async {
        // Сначала загружаем первое медиа с приоритетом
        if let firstMedia = pin.medias.first,
           let urlString = firstMedia.mediaURL?.absoluteString {
            let image = switch firstMedia.type {
            case .image:
                await ImageProvider.loadOrGetImage(for: urlString, .media)
            case .video:
                await ImageProvider.loadOrGetVideoThumbnail(for: urlString)
            }

            if let image {
                cachedImages[firstMedia.id] = image
            }
            isFirstMediaCached = true
        }

        // Потом загружаем остальные параллельно
        await withTaskGroup(of: (Int, UIImage?).self) { group in
            for media in pin.medias.dropFirst() {
                guard let urlString = media.mediaURL?.absoluteString else { continue }

                group.addTask {
                    let image = switch media.type {
                    case .image:
                        await ImageProvider.loadOrGetImage(for: urlString, .media)
                    case .video:
                        await ImageProvider.loadOrGetVideoThumbnail(for: urlString)
                    }
                    return (media.id, image)
                }
            }

            for await (id, image) in group {
                if let image {
                    cachedImages[id] = image
                }
            }
        }
    }
}
