import ProjectDescription

let project = Project(
    name: "PinzNetworking",
    targets: [
        .target(
            name: "PinzNetworking",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.hse.PinzNetworking",
            deploymentTargets: .iOS("18.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            dependencies: [
                .project(target: "PinzBase", path: "../PinzBase"),
                .project(target: "PinzDomain", path: "../PinzDomain"),
                .external(name: "Moya"),
                .external(name: "Vapor")
            ]
        )
    ]
)

