import ProjectDescription

let project = Project(
    name: "PinzNavigation",
    targets: [
        .target(
            name: "PinzNavigation",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.PinzNavigation",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: [
                .project(target: "PinzDomain", path: "../PinzDomain"),
                .project(target: "PinzAuthentication", path: "../PinzAuthentication"),
                .project(target: "PinzProfile", path: "../PinzProfile"),
                .project(target: "PinzTrips", path: "../PinzTrips"),
                .project(target: "PinzPins", path: "../PinzPins"),
                .project(target: "PinzFeed", path: "../PinzFeed")
            ]
        )
    ]
)
