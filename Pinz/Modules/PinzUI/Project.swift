import ProjectDescription

let project = Project(
    name: "PinzUI",
    targets: [
        .target(
            name: "PinzUI",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.PinzUI",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            resources: ["Resources/**"],
            dependencies: [
                .project(target: "PinzDomain", path: "../PinzDomain"),
            ]
        )
    ]
)

