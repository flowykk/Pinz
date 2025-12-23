import Foundation

public enum HelperFunctions {
    /// Cubic ease-in-out function for smooth animations
    /// - Parameter t: Progress value between 0 and 1
    /// - Returns: Eased progress value
    public static func easeInOutCubic(_ t: Double) -> Double {
        if t < 0.5 {
            return 4 * t * t * t
        } else {
            let f = 2 * t - 2
            return 0.5 * f * f * f + 1
        }
    }
}

