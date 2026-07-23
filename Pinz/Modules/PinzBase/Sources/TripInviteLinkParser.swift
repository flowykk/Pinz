import Foundation

public enum TripInviteLinkParser {

    private static let allowedHTTPSHosts: Set<String> = [
        "pinz.website",
        "www.pinz.website",
        "localhost",
        "127.0.0.1",
    ]

    public static func inviteToken(from url: URL) -> String? {
        guard let scheme = url.scheme?.lowercased() else { return nil }

        if scheme == "pinz" {
            return tokenFromCustomScheme(url)
        }

        if scheme == "https" || scheme == "http" {
            return tokenFromHTTP(url)
        }

        return nil
    }

    private static func tokenFromCustomScheme(_ url: URL) -> String? {
        guard url.host?.lowercased() == "join" else { return nil }
        let trimmed = url.path.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        guard !trimmed.isEmpty, !trimmed.contains("/") else { return nil }
        return trimmed.removingPercentEncoding ?? trimmed
    }

    private static func tokenFromHTTP(_ url: URL) -> String? {
        guard let host = url.host?.lowercased(), allowedHTTPSHosts.contains(host) else {
            return nil
        }
        let parts = url.path.split(separator: "/").map(String.init)
        guard let joinIndex = parts.firstIndex(of: "join"),
              joinIndex + 1 < parts.count
        else {
            return nil
        }
        let token = parts[joinIndex + 1]
        guard !token.isEmpty else { return nil }
        return token.removingPercentEncoding ?? token
    }
}
