import SwiftUI
import PinzBase
import PinzDomain

/// Shared “pins with issues” screen used by trip creation and pin upload review.
public struct PinProblemsListView: View {

    @State private var viewModel: PinProblemsListViewModel

    @Environment(\.appRouter) private var router

    private let rowTapBehavior: PinProblemsListRowTapBehavior

    public init(
        draftBinding: PinProblemsDraftBinding,
        pins: [Pin],
        rowTapBehavior: PinProblemsListRowTapBehavior = .opensPinInfo
    ) {
        self.rowTapBehavior = rowTapBehavior
        _viewModel = State(initialValue: PinProblemsListViewModel(draftBinding: draftBinding, pins: pins))
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: true) {
                header
            } content: {
                content
            }

            if viewModel.pinsWithIssues.isEmpty {
                VStack {
                    Spacer()
                    Text(PinzBaseStrings.TripCreationProblems.Message.noProblems)
                        .roundedFont(size: 15)
                        .foregroundStyle(PinzUIAsset.textSecondary.swiftUIColor)
                    Spacer()
                }
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear { viewModel.setRouter(router) }
    }

    @ViewBuilder
    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { viewModel.dispatch(.navigate(.back)) }
            )
        }, centerView: {
            HeaderTitle(PinzBaseStrings.TripCreationProblems.Title.main)
        })
    }

    @ViewBuilder
    private var content: some View {
        let pinsWithIssues = viewModel.pinsWithIssues

        ScrollView {
            VStack(spacing: 0) {
                ForEach(pinsWithIssues.indices, id: \.self) { index in
                    ProblemPinRow(
                        item: pinsWithIssues[index],
                        onTap: rowTapBehavior == .opensPinInfo
                            ? { viewModel.navigateToPinInfo(at: index, router: router) }
                            : nil
                    )
                    .padding(.top, 8)

                    if index != pinsWithIssues.count - 1 {
                        Divider().padding(.leading, 16)
                    }
                }
            }
            .padding(.bottom, 100)
        }
        .scrollIndicators(.hidden)
    }
}

private struct ProblemPinRow: View {
    let item: PinProblemsListViewModel.ProblemPin
    let onTap: (() -> Void)?

    var body: some View {
        Group {
            if let onTap {
                rowContent
                    .contentShape(Rectangle())
                    .onTapGesture(perform: onTap)
            } else {
                rowContent
            }
        }
    }

    private var rowContent: some View {
        VStack(alignment: .leading, spacing: 4) {
            HeaderPinShortInfo(pin: item.pin, isPrivacyShown: false)

            Text(item.issueText)
                .roundedFont(size: 14, foregroundColor: PinzUIAsset.accentOrange.swiftUIColor)
                .padding(.horizontal, 16)
                .padding(.bottom, 8)
        }
    }
}
