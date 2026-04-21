import SwiftUI
import PinzBase
import PinzDomain
import PinzUI
import PinzNetworking

public struct AddMediaHostView: View {
    @State private var viewModel: AddMediaHostViewModel
    @State private var selectionViewModel: AddMediaSelectionViewModel
    @State private var groupingViewModel: AddMediaGroupingViewModel?
    @State private var reviewViewModel: AddMediaReviewViewModel?

    @Environment(\.appRouter) private var router

    public init(tripId: String) {
        _viewModel = State(wrappedValue: AddMediaHostViewModel(tripId: tripId))
        _selectionViewModel = State(wrappedValue: AddMediaSelectionViewModel(tripId: tripId))
    }

    public var body: some View {
        Group {
            switch viewModel.step {
            case .selection:
                AddMediaSelectionView(
                    viewModel: selectionViewModel,
                    onBack: {
                        viewModel.dispatch(.navigate(.back))
                    },
                    onSessionReady: handleSessionReady
                )
            case .grouping:
                if let groupingViewModel {
                    AddMediaGroupingView(
                        viewModel: groupingViewModel,
                        onBack: {
                            resetToSelection()
                        },
                        onRetry: {
                            await processGrouping(for: groupingViewModel)
                        },
                        onContinue: {
                            openReview(with: groupingViewModel)
                        }
                    )
                } else {
                    AddMediaSelectionView(
                        viewModel: selectionViewModel,
                        onBack: {
                            viewModel.dispatch(.navigate(.back))
                        },
                        onSessionReady: handleSessionReady
                    )
                }
            case .review:
                if let reviewViewModel {
                    AddMediaReviewView(
                        viewModel: reviewViewModel,
                        onBack: {
                            resetToSelection()
                        },
                        onRetry: {
                            await applyReview(for: reviewViewModel)
                        },
                        onApply: {
                            await applyReview(for: reviewViewModel)
                        }
                    )
                } else {
                    AddMediaSelectionView(
                        viewModel: selectionViewModel,
                        onBack: {
                            viewModel.dispatch(.navigate(.back))
                        },
                        onSessionReady: handleSessionReady
                    )
                }
            }
        }
        .onAppear { viewModel.setRouter(router) }
    }

    private func handleSessionReady(
        _ session: AddMediaStartDTO,
        _ loadedMedia: [LoadedMedia]
    ) {
        guard !loadedMedia.isEmpty else {
            resetToSelection()
            return
        }

        let preparedGroupingVM = AddMediaGroupingViewModel(
            tripId: viewModel.tripId,
            session: session,
            loadedMedia: loadedMedia
        )
        groupingViewModel = preparedGroupingVM
        viewModel.dispatch(.openGrouping(session: session, loadedMedia: loadedMedia))
        Task { await processGrouping(for: preparedGroupingVM) }
    }

    private func processGrouping(for groupingVM: AddMediaGroupingViewModel) async {
        do {
            try await groupingVM.asyncDispatch(.startGrouping)
            await MainActor.run {
                viewModel.markGroupingState(
                    draftPins: groupingVM.draftPins,
                    existingMediaIds: Array(groupingVM.existingMediaIds),
                    existingPinsPreview: groupingVM.existingPinsPreview
                )
            }
        } catch {
            await MainActor.run {
                if isRestartNeeded(for: error) {
                    resetToSelection()
                } else {
                    viewModel.markGroupingFailed()
                }
            }
        }
    }

    private func openReview(with groupingVM: AddMediaGroupingViewModel) {
        guard viewModel.flowStatus == .ready, groupingVM.canProceed else {
            return
        }
        let draftPins = groupingVM.draftPins
        let existingMediaIds = Array(viewModel.existingMediaIds)
        let existingPinsPreview = viewModel.existingPinsPreview
        let deletedMediaIds = Array(groupingVM.deletedMediaIds)

        reviewViewModel = nil
        reviewViewModel = AddMediaReviewViewModel(
            tripId: viewModel.tripId,
            session: groupingVM.session,
            draftPins: draftPins,
            existingMediaIds: existingMediaIds,
            existingPinsPreview: existingPinsPreview,
            deletedMediaIds: deletedMediaIds
        )
        viewModel.markReviewState(
            draftPins: draftPins,
            existingMediaIds: existingMediaIds,
            existingPinsPreview: existingPinsPreview,
            deletedMediaIds: deletedMediaIds
        )
        viewModel.dispatch(.openReview)
    }

    private func applyReview(for reviewVM: AddMediaReviewViewModel) async {
        do {
            try await reviewVM.asyncDispatch(.apply)
            await MainActor.run {
                viewModel.dispatch(.finish)
            }
        } catch {
            await MainActor.run {
                if isRestartNeeded(for: error) {
                    resetToSelection()
                } else {
                    reviewVM.markFailed()
                }
            }
        }
    }

    private func isRestartNeeded(for error: Error) -> Bool {
        if let httpError = error as? HTTPError {
            return httpError == .conflict || httpError == .preconditionFailed
        }
        return false
    }

    private func resetToSelection() {
        viewModel.dispatch(.backToSelection)
        selectionViewModel.reset()
        groupingViewModel = nil
        reviewViewModel = nil
    }
}
