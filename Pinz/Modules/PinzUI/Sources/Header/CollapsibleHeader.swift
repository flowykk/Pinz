import SwiftUI

public struct CollapsibleHeader<
    Header: View,
    StickyHeader: View,
    Content: View
>: View {
    var spacing: CGFloat = 8
    @ViewBuilder var header: Header
    @ViewBuilder var stickyHeader: StickyHeader
    @ViewBuilder var content: Content
    let needsBlur: Bool

    @State private var currentDragOffset: CGFloat = 0
    @State private var previousDragOffset: CGFloat = 0
    @State private var headerOffset: CGFloat = 0
    @State private var headerSize: CGFloat = 0
    @State private var scrollOffset: CGFloat = 0

    public init(
        needsBlur: Bool = false,
        @ViewBuilder header: () -> Header = { EmptyView() },
        @ViewBuilder stickyHeader: () -> StickyHeader = { EmptyView() },
        @ViewBuilder content: () -> Content = { EmptyView() },
    ) {
        self.needsBlur = needsBlur
        self.header = header()
        self.stickyHeader = stickyHeader()
        self.content = content()
    }

    public var body: some View {
        ScrollView(.vertical) {
            content
        }
        .scrollIndicators(.hidden)
        .frame(maxWidth: .infinity)
        .onScrollGeometryChange(for: CGFloat.self, of: {
            $0.contentOffset.y + $0.contentInsets.top
        }, action: { oldValue, newValue in
            scrollOffset = newValue
        })
        .simultaneousGesture(
            DragGesture(minimumDistance: 10)
                .onChanged { value in
                    let dragOffset = -max(0, abs(value.translation.height) - 100) *
                        (value.translation.height < 0 ? -1 : 1)

                    previousDragOffset = currentDragOffset
                    currentDragOffset = dragOffset

                    let deltaOffset = ((currentDragOffset - previousDragOffset) * 0.7).rounded()

                    headerOffset = max(min(headerOffset + deltaOffset, headerSize), 0)
                }.onEnded { _ in
                    withAnimation(.easeInOut(duration: 0.3)) {
                        if headerOffset > (headerSize * 0.3) && scrollOffset > headerSize {
                            headerOffset = headerSize
                        } else {
                            headerOffset = 0
                        }
                    }

                    currentDragOffset = 0
                    previousDragOffset = 0
                }

        )
        .onChange(of: scrollOffset, { oldValue, newValue in
            if scrollOffset <= 10 {
                withAnimation(.easeInOut(duration: 0.3)) {
                    headerOffset = 0
                }
            }
        })
        .safeAreaInset(edge: .top, spacing: 0) {
            headerView
        }
    }

    @ViewBuilder
    private var headerView: some View {
        VStack(spacing: spacing) {
            header
                .onGeometryChange(for: CGFloat.self) {
                    $0.size.height
                } action: { newValue in
                    headerSize = newValue + spacing
                }

            stickyHeader
        }
        .padding(.bottom, 8)
        .offset(y: -headerOffset)
        .clipped()
        .background {
            VStack(spacing: 0) {
                Rectangle().fill(.ultraThinMaterial)
                Divider()
            }
            .overlay(
                Rectangle().fill(.white).ignoresSafeArea()
                    .if(needsBlur) { $0.opacity(overlayOpacity) }
            )
            .ignoresSafeArea()
            .offset(y: -headerOffset)
        }
        .contentShape(Rectangle())
        .onTapGesture {
            if headerOffset > 0 {
                withAnimation(.easeInOut(duration: 0.3)) {
                    headerOffset = 0
                }
            }
        }
    }
    
    private var overlayOpacity: Double {
        guard headerSize > 0 else { return 1.0 }
        let progress = headerOffset / headerSize
        return max(0, 1 - progress)
    }
}
