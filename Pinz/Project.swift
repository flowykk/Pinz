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
                .project(target: "PinzBase", path: "Modules/PinzBase"),
                .project(target: "PinzDomain", path: "Modules/PinzDomain"),
                .project(target: "PinzAuthentication", path: "Modules/PinzAuthentication"),
                .project(target: "PinzProfile", path: "Modules/PinzProfile"),
                .project(target: "PinzNetworking", path: "Modules/PinzNetworking"),
                .project(target: "PinzUI", path: "Modules/PinzUI"),
                .project(target: "PinzNavigation", path: "Modules/PinzNavigation"),
                .project(target: "PinzTrips", path: "Modules/PinzTrips"),
                .project(target: "PinzPins", path: "Modules/PinzPins"),
                .project(target: "PinzFeed", path: "Modules/PinzFeed")
            ],
            settings: .settings(
                base: [
                    "ASSETCATALOG_COMPILER_APPICON_NAME": "PinzLightP",
                    "ASSETCATALOG_COMPILER_INCLUDE_ALL_APPICON_ASSETS": "YES",
                ]
            )
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
