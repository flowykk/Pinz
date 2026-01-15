import ProjectDescription

let project = Project(
    name: "PinzNavigation",
    targets: [
        .target(
            name: "PinzNavigation",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.PinzNavigation",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: []
        )
    ]
)
