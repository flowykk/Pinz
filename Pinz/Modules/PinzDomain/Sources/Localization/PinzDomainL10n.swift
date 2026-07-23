import Foundation

enum PinzDomainL10n {
    static func string(_ key: String) -> String {
        String(localized: String.LocalizationValue(key), bundle: .module, locale: Locale.current)
    }
}
