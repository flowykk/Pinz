import SwiftUI
import PinzBase
import PinzDomain
import PinzUI
import PinzNetworking

public struct AddMediaHostView: View {
    @State private var viewModel: AddMediaHostViewModel
    @State private var selectionViewModel: AddMediaSelectionViewModel
    @State private var groupingViewModel: AddMediaGroupingViewModel?

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
                            await applyChanges(using: groupingViewModel)
                        },
                        onContinue: {
                            await applyChanges(using: groupingViewModel)
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

    private func applyChanges(using groupingVM: AddMediaGroupingViewModel) async {
        guard groupingVM.canProceed else {
            return
        }

        do {
            try await groupingVM.asyncDispatch(.applyGroupsAndProcess)
            await MainActor.run {
                viewModel.dispatch(.finish)
            }
        } catch {
            await MainActor.run {
                if isRestartNeeded(for: error) {
                    resetToSelection()
                } else {
                    groupingVM.setFailedState()
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
    }
}
