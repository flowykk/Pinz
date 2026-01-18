import ProjectDescription

let project = Project(
    name: "PinzTrips",
    targets: [
        .target(
            name: "PinzTrips",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.PinzTrips",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: [
                .project(target: "PinzBase", path: "../PinzBase"),
                .project(target: "PinzDomain", path: "../PinzDomain"),
                .project(target: "PinzNetworking", path: "../PinzNetworking"),
                .project(target: "PinzUI", path: "../PinzUI"),
            ]
        )
    ]
)
