import ProjectDescription

let project = Project(
    name: "PinzBase",
    targets: [
        .target(
            name: "PinzBase",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.PinzBase",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: [
                .project(target: "PinzDomain", path: "../PinzDomain"),
            ]
        )
    ]
)

