import ProjectDescription

let project = Project(
    name: "PinzProfile",
    targets: [
        .target(
            name: "PinzProfile",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.PinzProfile",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: [
                .project(target: "PinzUI", path: "../PinzUI")
            ]
        )
    ]
)
