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
                .project(target: "PinzUI", path: "../PinzUI"),
                .project(target: "PinzNavigation", path: "../PinzNavigation")
            ]
        )
    ]
)
