import ProjectDescription

let project = Project(
    name: "PinzProfile",
    targets: [
        .target(
            name: "PinzProfile",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.hse.PinzProfile",
            deploymentTargets: .iOS("18.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: [
                .project(target: "PinzBase", path: "../PinzBase"),
                .project(target: "PinzDomain", path: "../PinzDomain"),
                .project(target: "PinzNetworking", path: "../PinzNetworking"),
                .project(target: "PinzUI", path: "../PinzUI"),
                .project(target: "PinzAccessibility", path: "../PinzAccessibility"),
            ]
        )
    ]
)
