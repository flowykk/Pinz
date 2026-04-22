import SwiftUI
import PinzUI
import PinzBase

enum StorageSettingsIcon: String, Setting.Icon {
    case chevronRight = "chevron.right"
    case opticaldiscdrive = "opticaldiscdrive"
    case trash = "trash"
}

public struct StorageSettingsView: View {
    @AppStorage(FileManagerImageStorage.cacheEnabledKey) private var isImageCachingEnabled = true
    @State private var cacheSize = ""
    @State private var showClearCacheAlert = false

    @Environment(\.appRouter) private var router

    public init() { }

    public var body: some View {
        VStack(spacing: 0) {
            Header(leftView: {
                PinzButton(
                    type: .icon(.chevronLeft),
                    tint: PinzUIAsset.textPrimary.swiftUIColor,
                    action: .plain { router?.pop() }
                )
            }, centerView: {
                HeaderTitle(
                    PinzBaseStrings.Profile.Label.storage
                )
            })

            VStack(spacing: 12) {
                settingsSection

                Spacer()
            }
            .padding(.top, 8)
            .padding(.horizontal, 12)
        }
        .background(PinzUIAsset.background.swiftUIColor)
        .onAppear {
            FileManagerImageStorage.shared.isCachingEnabled = isImageCachingEnabled
            if isImageCachingEnabled {
                refreshCacheSize()
            } else {
                cacheSize = "0 B"
            }
        }
        .onChange(of: isImageCachingEnabled) { _, isEnabled in
            FileManagerImageStorage.shared.isCachingEnabled = isEnabled

            if !isEnabled {
                FileManagerImageStorage.shared.clear()
                cacheSize = "0 B"
            } else {
                refreshCacheSize()
            }
        }
        .alert(PinzBaseStrings.Alert.ClearCache.title, isPresented: $showClearCacheAlert) {
            Button(PinzBaseStrings.Common.Button.cancel, role: .cancel) { }
            Button(PinzBaseStrings.Alert.ClearCache.confirm, role: .destructive) {
                FileManagerImageStorage.shared.clear()
                cacheSize = "0 B"
            }
        } message: {
            Text(PinzBaseStrings.Alert.ClearCache.message)
        }
    }

    private var settingsSection: some View {
        VStack(spacing: 12) {
            SettingsGroup(
                settings: [
                    .toggle(Setting.ToggleSetting(
                        id: "storageImageCaching",
                        leading: .iconTitle(
                            StorageSettingsIcon.opticaldiscdrive,
                            PinzBaseStrings.Profile.Label.saveImagesLocally
                        ),
                        value: $isImageCachingEnabled
                    ))
                ],
                subtitle: PinzBaseStrings.Profile.Hint.saveImagesLocally
            )

            SettingsGroup(
                settings: [
                    .default(Setting.DefaultSetting(
                        id: "storageClearCache",
                        leading: .iconTitle(
                            StorageSettingsIcon.trash,
                            PinzBaseStrings.Profile.Label.clearCache
                        ),
                        trailing: .valuesIcon([.text(cacheSize)], StorageSettingsIcon.chevronRight),
                        action: .plain { showClearCacheAlert = true }
                    )),
                ]
            )
        }
    }

    private func refreshCacheSize() {
        FileManagerImageStorage.shared.isCachingEnabled = isImageCachingEnabled
        cacheSize = FileManagerImageStorage.shared.getCacheSize()
    }
}

#if DEBUG
#Preview {
    StorageSettingsView()
}
#endif
