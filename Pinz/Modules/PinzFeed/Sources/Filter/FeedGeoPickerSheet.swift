import SwiftUI
import PinzUI
import PinzBase

struct FeedGeoPickerSheet: View {

    enum CatalogSegment: Sendable {
        case countries
        case cities
    }

    let title: String
    let segment: CatalogSegment
    @Binding var selectedSlug: String
    @Binding var isPresented: Bool

    @State private var entries: [FeedGeoCatalog.Entry] = []
    @State private var search = ""

    private var filteredEntries: [FeedGeoCatalog.Entry] {
        let q = search.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !q.isEmpty else { return entries }
        return entries.filter { entry in
            FeedGeoCatalog.displayLabel(for: entry).localizedStandardContains(q)
                || entry.localized.ru.localizedStandardContains(q)
                || entry.localized.eng.localizedStandardContains(q)
                || entry.key.localizedStandardContains(q.lowercased())
        }
    }

    var body: some View {
        NavigationStack {
            Group {
                if entries.isEmpty {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    List(filteredEntries) { entry in
                        Button {
                            selectedSlug = entry.key
                            isPresented = false
                        } label: {
                            HStack {
                                Text(FeedGeoCatalog.displayLabel(for: entry))
                                    .roundedFont(size: 16, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                                Spacer()
                                if entry.key == selectedSlug {
                                    Image(systemName: "checkmark")
                                        .roundedFont(size: 14, foregroundColor: PinzUIAsset.accentGreen.swiftUIColor)
                                }
                            }
                        }
                    }
                    .listStyle(.plain)
                    .searchable(text: $search, placement: .navigationBarDrawer(displayMode: .always))
                }
            }
            .navigationTitle(title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(PinzBaseStrings.Common.Button.cancel) {
                        isPresented = false
                    }
                }
                ToolbarItem(placement: .primaryAction) {
                    Button(PinzBaseStrings.Feed.Filter.clearLocation) {
                        selectedSlug = ""
                        isPresented = false
                    }
                    .disabled(selectedSlug.isEmpty)
                }
            }
            .task {
                let loaded = await Task.detached { () -> [FeedGeoCatalog.Entry] in
                    switch segment {
                    case .countries: return FeedGeoCatalog.allCountries()
                    case .cities: return FeedGeoCatalog.allCities()
                    }
                }.value
                entries = loaded
            }
        }
    }
}
