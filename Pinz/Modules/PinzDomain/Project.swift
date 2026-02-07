import ProjectDescription

let project = Project(
    name: "PinzDomain",
    targets: [
        .target(
            name: "PinzDomain",
            destinations: .iOS,
            product: .framework,
            bundleId: "io.tuist.PinzDomain",
            deploymentTargets: .iOS("18.0"),
            infoPlist: .default,
            sources: ["Sources/**"],
            resources: ["Resources/**"],
            dependencies: []
        )
    ]
)

