import ProjectDescription

let project = Project(
    name: "Pinz",
    targets: [
        .target(
            name: "Pinz",
            destinations: .iOS,
            product: .app,
            bundleId: "io.tuist.Pinz",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .extendingDefault(
                with: [
                    "UILaunchScreen": [
                        "UIColorName": "",
                        "UIImageName": "",
                    ],
                ]
            ),
            sources: ["Pinz/Sources/**"],
            resources: ["Pinz/Resources/**"],
            dependencies: [
                .project(target: "Base", path: "Modules/Base"),
                .project(target: "Features", path: "Modules/Features"),
                .project(target: "UIComponents", path: "Modules/UIComponents")
            ]
        ),
        .target(
            name: "PinzTests",
            destinations: .iOS,
            product: .unitTests,
            bundleId: "io.tuist.PinzTests",
            deploymentTargets: .iOS("17.0"),
            infoPlist: .default,
            sources: ["Pinz/Tests/**"],
            resources: [],
            dependencies: [.target(name: "Pinz")]
        )
    ]
)
