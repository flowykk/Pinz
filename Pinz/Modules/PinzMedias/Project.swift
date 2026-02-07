import ProjectDescription

let project = Project(
    name: "PinzMedias",
    targets: [
        .target(
            name: "PinzMedias",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.PinzMedias",
            deploymentTargets: .iOS("18.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: [
                .project(target: "PinzBase", path: "../PinzBase"),
                .project(target: "PinzDomain", path: "../PinzDomain"),
                .project(target: "PinzUI", path: "../PinzUI"),
            ]
        )
    ]
)
