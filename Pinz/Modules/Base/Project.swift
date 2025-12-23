import ProjectDescription

let project = Project(
    name: "Base",
    targets: [
        .target(
            name: "Base",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.Base",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: []
        )
    ]
)

