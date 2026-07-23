import ProjectDescription

let project = Project(
    name: "PinzBase",
    targets: [
        .target(
            name: "PinzBase",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.hse.PinzBase",
            deploymentTargets: .iOS("18.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            resources: ["Resources/**"],
            dependencies: [
                .project(target: "PinzDomain", path: "../PinzDomain"),
            ]
        )
    ]
)

