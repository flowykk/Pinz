import ProjectDescription

let project = Project(
    name: "PinzFeed",
    targets: [
        .target(
            name: "PinzFeed",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.PinzFeed",
            deploymentTargets: .iOS("18.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            resources: ["Resources/**"],
            dependencies: [
                .project(target: "PinzBase", path: "../PinzBase"),
                .project(target: "PinzDomain", path: "../PinzDomain"),
                .project(target: "PinzNetworking", path: "../PinzNetworking"),
                .project(target: "PinzUI", path: "../PinzUI"),
            ]
        )
    ]
)
