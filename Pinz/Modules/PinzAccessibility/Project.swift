import ProjectDescription

let project = Project(
    name: "PinzAccessibility",
    targets: [
        .target(
            name: "PinzAccessibility",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.PinzAccessibility",
            deploymentTargets: .iOS("18.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: []
        )
    ]
)
