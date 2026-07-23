import SwiftUI
import PinzUI

struct RecommendationPostCard<Content: View>: View {

    private let isRecommended: Bool
    private let badge: String?
    private let content: () -> Content

    init(
        isRecommended: Bool,
        badge: String?,
        @ViewBuilder content: @escaping () -> Content
    ) {
        self.isRecommended = isRecommended
        self.badge = badge
        self.content = content
    }

    var body: some View {
        let card = content().padding(.bottom, isRecommended ? 8 : 0)
        if isRecommended {
            card
                .background(
                    RoundedRectangle(cornerRadius: 14)
                        .fill(
                            LinearGradient(
                                colors: [
                                    Color.purple.opacity(0.35),
                                    Color.purple.opacity(0.16),
                                    .clear
                                ],
                                startPoint: .topLeading,
                                endPoint: .bottomTrailing
                            )
                            .opacity(1)
                        )
                )
                .overlay(
                    RoundedRectangle(cornerRadius: 14)
                        .stroke(
                            LinearGradient(
                                colors: [Color.white.opacity(0.5), Color.purple.opacity(0.5)],
                                startPoint: .topLeading,
                                endPoint: .bottomTrailing
                            ),
                            lineWidth: 1.2
                        )
                )
                .shadow(
                    color: Color.purple.opacity(0.35),
                    radius: 12,
                    x: 0,
                    y: 6
                )
                .overlay(alignment: .bottomLeading) {
                    if let badge {
                        let badgeBottomInset: CGFloat = 60
                        Text(badge)
                            .roundedFont(size: 11, weight: .semibold, foregroundColor: .white)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 6)
                            .background(
                                Capsule()
                                    .fill(
                                        LinearGradient(
                                            colors: [Color.purple.opacity(0.95), Color.indigo.opacity(0.85)],
                                            startPoint: .topLeading,
                                            endPoint: .bottomTrailing
                                        )
                                    )
                            )
                            .overlay(
                                Capsule()
                                    .stroke(Color.white.opacity(0.35), lineWidth: 1)
                            )
                            .padding(.leading, 8)
                            .padding(.bottom, badgeBottomInset)
                    }
                }
                .overlay(
                    Circle()
                        .fill(Color.purple.opacity(0.07))
                        .blur(radius: 60)
                        .scaleEffect(1.3)
                        .frame(width: 250, height: 250)
                        .offset(x: 150, y: -120),
                    alignment: .topTrailing
                )
                .overlay(
                    Circle()
                        .fill(Color.purple.opacity(0.07))
                        .blur(radius: 60)
                        .scaleEffect(1.3)
                        .frame(width: 220, height: 220)
                        .offset(x: -120, y: 110),
                    alignment: .bottomLeading
                )
        } else {
            card
        }
    }
}
