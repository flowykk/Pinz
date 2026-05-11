import SwiftUI
import CoreLocation
import PinzUI
import PinzDomain
import PinzBase

private enum PinUploadReviewIcon: String, Setting.Icon {
    case info = "info.circle.fill"
    case calendar = "calendar"
    case mappin = "mappin.circle.fill"
}

public struct PinUploadReviewView: View {

    @State private var viewModel: PinUploadReviewViewModel

    @State private var isCategoryPickerPresented = false
    @State private var isStartDatePickerPresented = false
    @State private var isEndDatePickerPresented = false
    @State private var datePickerHeight: CGFloat = 0
    @State private var showCancelConfirmation = false

    let galleryColumns: Int = 3
    let gallerySpacing: CGFloat = 4

    @Environment(\.appRouter) private var router
    @Environment(\.showToast) private var showToast

    public init(tripId: String, sessionId: String) {
        viewModel = PinUploadReviewViewModel(tripId: tripId, sessionId: sessionId)
    }

    public var body: some View {
        ZStack {
            CollapsibleHeader(needsBlur: viewModel.state == .gallery) {
                header
            } stickyHeader: {
                SegmentedPicker(selection: $viewModel.state, items: [.info, .gallery])
                    .padding(.horizontal, 12)
            } content: {
                Group {
                    if viewModel.state == .info {
                        infoContent
                    } else {
                        gallery
                    }
                }
                .padding(.bottom, 60)
            }

            if viewModel.isLoading && !viewModel.initialLoaded {
                LoadingView()
            } else if viewModel.initialLoaded {
                gradientWithButtons
            }
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .navigationBarBackButtonHidden(true)
        .onAppear {
            viewModel.setRouter(router)
            viewModel.setShowToast(showToast)
            Task { try? await viewModel.asyncDispatch(.reload) }
        }
        .itemsPickerSheet(
            isPresented: $isCategoryPickerPresented,
            items: PinCategory.allCases,
            selection: $viewModel.category,
            customizableItem: .custom(),
            saveCustomizableItem: { value in
                viewModel.category = .custom(value)
            }
        )
        .datePickerSheet(
            isPresented: $isStartDatePickerPresented,
            date: $viewModel.startDate,
            pickerHeight: $datePickerHeight
        )
        .datePickerSheet(
            isPresented: $isEndDatePickerPresented,
            date: $viewModel.endDate,
            pickerHeight: $datePickerHeight
        )
        .confirmationDialog(
            "Отменить создание пина?",
            isPresented: $showCancelConfirmation,
            titleVisibility: .visible
        ) {
            Button("Отменить создание пина", role: .destructive) {
                Task { try? await viewModel.asyncDispatch(.cancel) }
            }
            Button("Продолжить", role: .cancel) {}
        }
    }

    @ViewBuilder
    private var header: some View {
        Header(leftView: {
            PinzButton(
                type: .icon(.chevronLeft),
                tint: PinzUIAsset.textPrimary.swiftUIColor,
                action: .plain { showCancelConfirmation = true }
            )
        }, centerView: {
            AnimatableHeaderTitle(
                animatableTitle: "Новый пин",
                title: $viewModel.name
            )
        }, rightView: {
            EmptyView()
        })
    }

    private var infoContent: some View {
        ScrollView {
            VStack(spacing: 12) {
                general
                tags
                descriptionEditing
            }
            .padding(.horizontal, 12)
        }
        .scrollIndicators(.hidden)
    }

    @ViewBuilder
    private var general: some View {
        SettingsGroup(
            title: PinzBaseStrings.PinCreation.Header.general,
            settings: [
                .textField(Setting.TextFieldSetting(
                    id: "pinNameTextField",
                    text: $viewModel.name,
                    placeholder: PinzBaseStrings.PinCreation.Placeholder.name
                )),
            ],
            subtitle: PinzBaseStrings.PinCreation.Hint.pinNameRules
        )

        SettingsGroup(
            settings: [
                .picker(Setting.PickerSetting(
                    id: "pinCategoryPicker",
                    leading: .iconTitle(PinUploadReviewIcon.info, viewModel.category.value),
                    isPickerPresented: $isCategoryPickerPresented
                )),
                .picker(Setting.PickerSetting(
                    id: "pinStartDatePicker",
                    leading: .iconTitle(PinUploadReviewIcon.calendar, PinzBaseStrings.Common.Label.startDate),
                    value: .text(viewModel.startDate?.formattedToDayMonthYear ?? PinzBaseStrings.Common.Label.notSelected),
                    isPickerPresented: $isStartDatePickerPresented
                )),
                .picker(Setting.PickerSetting(
                    id: "pinEndDatePicker",
                    leading: .iconTitle(PinUploadReviewIcon.calendar, PinzBaseStrings.Common.Label.endDate),
                    value: .text(viewModel.endDate?.formattedToDayMonthYear ?? PinzBaseStrings.Common.Label.notSelected),
                    isPickerPresented: $isEndDatePickerPresented
                )),
            ]
        )

        if let coords = viewModel.coordinates {
            coordinatesRow(coords: coords)
        }
    }

    private func coordinatesRow(coords: CLLocationCoordinate2D) -> some View {
        HStack(spacing: 8) {
            Image(systemName: PinUploadReviewIcon.mappin.rawValue)
                .frame(20)
                .foregroundColor(PinzUIAsset.textPrimary.swiftUIColor)
            VStack(alignment: .leading, spacing: 2) {
                Text(String(format: "%.5f, %.5f", coords.latitude, coords.longitude))
                    .roundedFont(size: 14, foregroundColor: PinzUIAsset.textPrimary.swiftUIColor)
                Text("из EXIF")
                    .roundedFont(size: 12, foregroundColor: PinzUIAsset.textSecondary.swiftUIColor)
            }
            Spacer()
        }
        .padding(12)
        .background(PinzUIAsset.backgroundSecondary.swiftUIColor)
        .cornerRadius(14)
    }

    private var tags: some View {
        VStack(alignment: .leading, spacing: 0) {
            SettingTitle(PinzBaseStrings.PinCreation.Header.tags)
                .padding(.leading, 12)
            TagsView(
                tags: viewModel.tags,
                onTagAdd: { tag in viewModel.dispatch(.addTag(tag)) },
                onTagDelete: { tag in viewModel.dispatch(.deleteTag(tag)) },
                style: .editable,
                maxTags: PinUploadReviewViewModel.maxTagsCount
            )
            .padding(.top, 2)
        }
    }

    private var descriptionEditing: some View {
        DescriptionEditingView(
            title: PinzBaseStrings.Common.Label.description,
            text: Binding(
                get: { viewModel.description ?? "" },
                set: { viewModel.description = $0 }
            ),
            placeholder: PinzBaseStrings.PinCreation.Placeholder.description
        )
    }

    private var gallery: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: gallerySpacing) {
                PinzGrid(
                    $viewModel.medias,
                    columns: galleryColumns,
                    spacing: gallerySpacing
                ) { media, _ in
                    galleryTile(for: media)
                }
            }
        }
        .scrollIndicators(.hidden)
        .padding(.horizontal, gallerySpacing)
    }

    @ViewBuilder
    private func galleryTile(for media: ReviewPinMediaDTO) -> some View {
        let isMarkedForDelete = viewModel.mediaToDelete.contains(media.mediaId)
        ZStack {
            LoadableImageThumbnail(url: URL(string: media.url)) { state in
                tileBody(state: state)
            }
            if isMarkedForDelete {
                RoundedRectangle(cornerRadius: 14)
                    .fill(Color.black.opacity(0.55))
                Image(systemName: "checkmark.circle.fill")
                    .font(.system(size: 28, weight: .bold))
                    .foregroundColor(.white)
            }
        }
        .overlay(alignment: .topTrailing) {
            Button {
                withAnimation(.easeInOut(duration: 0.2)) {
                    viewModel.dispatch(.toggleDeleteMedia(media.mediaId))
                }
            } label: {
                BadgeView(
                    icon: .trash,
                    color: isMarkedForDelete
                        ? PinzUIAsset.textSecondary.swiftUIColor
                        : PinzUIAsset.accentRed.swiftUIColor
                )
            }
            .padding(4)
        }
    }

    @ViewBuilder
    private func tileBody(state: LoadableMediaState) -> some View {
        switch state {
        case .empty:
            Rectangle()
                .fill(Color.gray.opacity(0.3))
                .aspectRatio(1, contentMode: .fit)
                .overlay { ProgressView().tint(.white) }
                .cornerRadius(14)
        case .ready(let image):
            Image(uiImage: image)
                .resizable()
                .scaledToFill()
                .frame(maxWidth: .infinity)
                .clipped()
                .cornerRadius(14)
        case .failure:
            Rectangle()
                .fill(Color.red.opacity(0.3))
                .aspectRatio(1, contentMode: .fit)
                .overlay {
                    Image(systemName: "exclamationmark.triangle.fill")
                        .foregroundColor(.white)
                }
                .cornerRadius(14)
        }
    }

    private var gradientWithButtons: some View {
        BottomGradientWithButtons {
            PinzButton(
                type: .slot(style: .primary, title: PinzBaseStrings.Common.Button.save),
                tint: PinzUIAsset.backgroundSecondary.swiftUIColor,
                action: .async { try await viewModel.asyncDispatch(.finalize) }
            )
            .disabledWithOpacity(saveButtonDisabled)
        }
    }

    private var saveButtonDisabled: Bool {
        let remaining = viewModel.medias.count - viewModel.mediaToDelete.count
        return remaining <= 0 || viewModel.isLoading
    }
}
