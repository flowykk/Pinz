import ProjectDescription

let project = Project(
    name: "Networking",
    targets: [
        .target(
            name: "Networking",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.Networking",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: [
                .project(target: "Base", path: "../Base"),
                .external(name: "Moya")
            ]
        )
    ]
)

