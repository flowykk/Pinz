import SwiftUI
import PhotosUI
import PinzBase
import PinzNetworking
import PinzDomain

@MainActor @Observable
final class AddMediaSelectionViewModel {

    enum LoadingStatus {
        case starting

        var localizedValue: String {
            switch self {
            case .starting:
                PinzBaseStrings.TripCreation.Loading.uploadingMedia
            }
        }
    }

    let tripId: String
    private(set) var isLoading = false
    private(set) var loadingStatus: LoadingStatus?
    private(set) var session: AddMediaStartDTO?
    private(set) var loadedMedia: [LoadedMedia] = []

    private let networkService = NetworkService.shared

    init(tripId: String) {
        self.tripId = tripId
    }

    func asyncDispatch(_ intent: AsyncIntent) async {
        switch intent {
        case let .startSession(items):
            await startSession(with: items)
        }
    }

    private func startSession(with pickerItems: [PhotosPickerItem]) async {
        changeLoading(to: true, status: .starting)
        session = nil

        let loadedMediaItems = await loadMedia(from: pickerItems)
        self.loadedMedia = loadedMediaItems
        let filesToUpload = loadedMediaItems.compactMap { media -> FileToUploadDTO? in
            if case .loading = media.content {
                return nil
            }
            return FileToUploadDTO(clientId: media.id.uuidString, contentType: media.uploadContentType)
        }

        guard !filesToUpload.isEmpty else {
            changeLoading(to: false, status: nil)
            self.loadedMedia = []
            return
        }

        do {
            session = try await networkService.addMediaStart(
                tripId: tripId,
                filesToUpload: filesToUpload
            )
            changeLoading(to: false, status: nil)
        } catch {
            changeLoading(to: false, status: nil)
            self.loadedMedia = []
            session = nil
        }
    }

    func reset() {
        withAnimation(.easeInOut(duration: 0.3)) {
            loadedMedia = []
            session = nil
            isLoading = false
            loadingStatus = nil
        }
    }

    private func loadMedia(from pickerItems: [PhotosPickerItem]) async -> [LoadedMedia] {
        let placeholderIds = pickerItems.map { _ in UUID() }
        let placeholders = placeholderIds.map { id in LoadedMedia(id: id, content: .loading) }

        var loadedMedia: [LoadedMedia] = []
        await withTaskGroup(of: (UUID, LoadedMedia?).self) { group in
            for index in pickerItems.indices {
                let id = placeholderIds[index]
                let pickerItem = pickerItems[index]
                group.addTask {
                    return (id, await MediaLoader.shared.load(from: pickerItem, id: id))
                }
            }

            for await result in group {
                if let media = result.1 {
                    loadedMedia.append(media)
                }
            }
        }

        if loadedMedia.count < pickerItems.count {
            let loadedIds = Set(loadedMedia.map(\.id))
            let failedCount = pickerItems.count - loadedIds.count
            if failedCount > 0 {
                loadedMedia += placeholders.filter { !loadedIds.contains($0.id) }
            }
        }

        return loadedMedia
    }

    private func changeLoading(to isLoading: Bool, status: LoadingStatus?) {
        withAnimation(.easeInOut(duration: 0.3)) {
            self.isLoading = isLoading
            self.loadingStatus = status
        }
    }
}

extension AddMediaSelectionViewModel {
    enum AsyncIntent {
        case startSession(items: [PhotosPickerItem])
    }
}
