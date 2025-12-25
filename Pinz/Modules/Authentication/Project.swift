import ProjectDescription

let project = Project(
    name: "Authentication",
    targets: [
        .target(
            name: "Authentication",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.Authentication",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: [
                .project(target: "Base", path: "../Base"),
                .project(target: "Networking", path: "../Networking"),
                .project(target: "UIComponents", path: "../UIComponents")
            ]
        )
    ]
)

