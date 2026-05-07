import Foundation

public enum PinzElement {
    public enum Trip {
        case button(Button)
        case input(Input)
        case row(Row)

        public enum Button: String {
            case openProfile
            case openTripInfo
            case edit
            case cancel
            case done
            case leave
            case delete
        }

        public enum Input {
            case name
            case description
            case season
            case category
            case dates
            case seasonPicker
            case categoryPicker
            case startDatePicker
            case endDatePicker

            var rawValue: String {
                switch self {
                case .name:
                    "name"
                case .description:
                    "description"
                case .season:
                    "season"
                case .category:
                    "category"
                case .dates:
                    "dates"
                case .seasonPicker:
                    "seasonPicker"
                case .categoryPicker:
                    "categoryPicker"
                case .startDatePicker:
                    "startDatePicker"
                case .endDatePicker:
                    "endDatePicker"
                }
            }
        }

        public enum Row {
            case headerTitle
            case headerTitleDetail
            case description
            case pins

            var rawValue: String {
                switch self {
                case .headerTitle:
                    "headerTitle"
                case .headerTitleDetail:
                    "headerTitleDetail"
                case .description:
                    "description"
                case .pins:
                    "pins"
                }
            }
        }

        var rawValue: String {
            switch self {
            case let .button(button):
                return "button.\(button.rawValue)"
            case let .input(input):
                return "input.\(input.rawValue)"
            case let .row(row):
                return "row.\(row.rawValue)"
            }
        }
    }

    public enum Profile {
        case button(Button)
        case input(Input)
        case row(Row)

        public enum Button: String {
            case edit
            case save
            case cancel
            case done
            case back
        }

        public enum Input {
            case nickname
            case email
            case verificationCode(Int = -1)

            var rawValue: String {
                switch self {
                case .nickname:
                    "nickname"
                case .email:
                    "email"
                case let .verificationCode(index):
                    index < 0 ? "verificationCode" : "verificationCode.\(index)"
                }
            }
        }

        public enum Row: String {
            case email
            case deleteAccount
            case headerNickname
        }

        var rawValue: String {
            switch self {
            case let .button(button):
                return "button.\(button.rawValue)"
            case let .input(input):
                return "input.\(input.rawValue)"
            case let .row(row):
                return "row.\(row.rawValue)"
            }
        }
    }

    public enum SettingsRow: String {
        case profileStats
        case profileTrips
        case profileWishlist
        case profileSavedMaps
        case profileStorage
        case profileNotifications
        case profileAppearance
        case profileLeave
        case profileDelete
        case profileEmail
    }

    public enum Wishlist {
        case button(Button)
        case input(Input)
        case row(Row)

        public enum Button: String {
            case add
            case back
            case edit
            case save
            case cancel
            case done
            case photo
            case delete
        }

        public enum Input {
            case name
            case description

            var rawValue: String {
                switch self {
                case .name:
                    "name"
                case .description:
                    "description"
                }
            }
        }

        public enum Row {
            case item(String)

            var rawValue: String {
                switch self {
                case let .item(id):
                    "item.\(id)"
                }
            }
        }

        var rawValue: String {
            switch self {
            case let .button(button):
                "button.\(button.rawValue)"
            case let .input(input):
                "input.\(input.rawValue)"
            case let .row(row):
                "row.\(row.rawValue)"
            }
        }
    }

    case trip(Trip)
    case pin(Pin)
    case profile(Profile)
    case settings(SettingsRow)
    case wishlist(Wishlist)

    public var accessibilityID: String {
        switch self {
        case let .trip(trip):
            return "trip.\(trip.rawValue)"
        case let .pin(pin):
            return "pin.\(pin.rawValue)"
        case let .profile(profile):
            return "profile.\(profile.rawValue)"
        case let .settings(setting):
            return "settings.\(setting.rawValue)"
        case let .wishlist(wishlist):
            return "wishlist.\(wishlist.rawValue)"
        }
    }

    public static func setting(for id: String) -> PinzElement? {
        if let setting = SettingsRow(rawValue: id) {
            return .settings(setting)
        }

        if let setting = SettingsRow(rawValue: id.trimmingPrefix("settings.")) {
            return .settings(setting)
        }

        switch id {
        case "nicknameTextField":
            return .profile(.input(.nickname))
        case "emailTextField":
            return .profile(.input(.email))
        case "headerNickname":
            return .profile(.row(.headerNickname))
        case "wishlistElementNameTextField":
            return .wishlist(.input(.name))
        case "wishlistElementPhoto":
            return .wishlist(.button(.photo))
        case "wishlistElementDelete":
            return .wishlist(.button(.delete))
        case "descriptionEditingTextField":
            return .wishlist(.input(.description))
        case "tripNameTextField":
            return .trip(.input(.name))
        case "tripDescriptionEditingTextField":
            return .trip(.input(.description))
        case "tripSeason":
            return .trip(.input(.season))
        case "tripCategory":
            return .trip(.input(.category))
        case "tripDates":
            return .trip(.input(.dates))
        case "tripSeasonPicker":
            return .trip(.input(.seasonPicker))
        case "tripCategoryPicker":
            return .trip(.input(.categoryPicker))
        case "tripStartDatePicker":
            return .trip(.input(.startDatePicker))
        case "tripEndDatePicker":
            return .trip(.input(.endDatePicker))
        case "tripPins":
            return .trip(.row(.pins))
        case "pinNameTextField":
            return .pin(.input(.name))
        case "pinDescriptionEditingTextField":
            return .pin(.input(.description))
        case "pinDelete":
            return .pin(.button(.delete))
        case "profileEmail":
            return .settings(.profileEmail)
        case "tripLeave":
            return .trip(.button(.leave))
        case "tripDelete":
            return .trip(.button(.delete))
        default:
            if id.hasPrefix("wishlist.row.item.") {
                let value = String(id.dropFirst("wishlist.row.item.".count))
                return .wishlist(.row(.item(value)))
            }

            if id.hasPrefix("profile.input.verificationCode.") {
                let parts = id.split(separator: ".")
                if let index = parts.last.flatMap({ Int($0) }) {
                    return .profile(.input(.verificationCode(index)))
                }
            }

            return nil
        }
    }
}

public extension PinzElement {
    enum Pin {
        case button(Button)
        case input(Input)
        case row(Row)

        public enum Button: String {
            case edit
            case cancel
            case done
            case delete
        }

        public enum Input: String {
            case name
            case description
        }

        public enum Row: String {
            case headerTitleDetail
        }

        var rawValue: String {
            switch self {
            case let .button(button):
                return "button.\(button.rawValue)"
            case let .input(input):
                return "input.\(input.rawValue)"
            case let .row(row):
                return "row.\(row.rawValue)"
            }
        }
    }
}

private extension String {
    func trimmingPrefix(_ prefix: String) -> String {
        hasPrefix(prefix)
            ? String(dropFirst(prefix.count))
            : self
    }
}
