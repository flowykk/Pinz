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
    @State private var cacheSize = "0 B"
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
            refreshCacheSize()
        }
        .onChange(of: isImageCachingEnabled) { _, isEnabled in
            FileManagerImageStorage.shared.isCachingEnabled = isEnabled

            if !isEnabled {
                FileManagerImageStorage.shared.clear()
                refreshCacheSize()
            } else {
                refreshCacheSize()
            }
        }
        .alert(PinzBaseStrings.Alert.ClearCache.title, isPresented: $showClearCacheAlert) {
            Button(PinzBaseStrings.Common.Button.cancel, role: .cancel) { }
            Button(PinzBaseStrings.Alert.ClearCache.confirm, role: .destructive) {
                clearCache()
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
                        action: .plain {
                            showClearCacheAlert = true
                        }
                    )),
                ]
            )
        }
    }

    private func clearCache() {
        FileManagerImageStorage.shared.clear()
        refreshCacheSize()
    }

    private func refreshCacheSize() {
        if isImageCachingEnabled {
            cacheSize = FileManagerImageStorage.shared.getCacheSize()
        } else {
            cacheSize = "0 B"
        }
    }
}

#if DEBUG
#Preview {
    StorageSettingsView()
}
#endif
