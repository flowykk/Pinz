import ProjectDescription

let project = Project(
    name: "PinzAuthentication",
    targets: [
        .target(
            name: "PinzAuthentication",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.PinzAuthentication",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: [
                .project(target: "PinzBase", path: "../PinzBase"),
                .project(target: "PinzDomain", path: "../PinzDomain"),
                .project(target: "PinzNetworking", path: "../PinzNetworking"),
                .project(target: "PinzNavigation", path: "../PinzNavigation"),
                .project(target: "PinzUI", path: "../PinzUI"),
            ]
        )
    ]
)

