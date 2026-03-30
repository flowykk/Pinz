import SwiftUI

public struct LoadingView: View {

    public init() {}

    public var body: some View {
        Spacer(minLength: 0)
        ProgressView()
        Spacer(minLength: 0)
    }
}
