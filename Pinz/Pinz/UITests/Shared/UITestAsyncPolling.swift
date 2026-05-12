import Foundation
import XCTest

extension XCTestCase {
    @MainActor
    func waitUntil(
        timeout: TimeInterval = 2.0,
        pollInterval: Duration = .milliseconds(100),
        _ condition: @MainActor () async -> Bool
    ) async -> Bool {
        let deadline = Date().addingTimeInterval(timeout)

        while Date() < deadline {
            if await condition() {
                return true
            }
            try? await Task.sleep(for: pollInterval)
        }

        return await condition()
    }
}
