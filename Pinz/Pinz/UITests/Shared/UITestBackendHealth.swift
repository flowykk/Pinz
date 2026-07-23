import Foundation
import XCTest

extension XCTestCase {
    func waitForBackendHealth(
        timeout: TimeInterval = 2.0,
        url: URL = URL(string: "http://localhost:8080/health")!
    ) -> Bool {
        let deadline = Date().addingTimeInterval(timeout)

        while Date() < deadline {
            if isBackendHealthy(url: url) {
                return true
            }
            Thread.sleep(forTimeInterval: 0.1)
        }
        return false
    }

    private func isBackendHealthy(url: URL) -> Bool {
        var request = URLRequest(url: url)
        request.httpMethod = "GET"
        request.timeoutInterval = 0.25

        let semaphore = DispatchSemaphore(value: 0)
        var isHealthy = false

        let task = URLSession.shared.dataTask(with: request) { _, response, _ in
            defer { semaphore.signal() }
            guard let response = response as? HTTPURLResponse else {
                return
            }
            isHealthy = (200 ... 299).contains(response.statusCode)
        }

        task.resume()
        _ = semaphore.wait(timeout: .now() + 0.3)
        return isHealthy
    }
}
