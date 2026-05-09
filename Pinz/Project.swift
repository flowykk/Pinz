import ProjectDescription

let project = Project(
    name: "Pinz",
    targets: [
        .target(
            name: "Pinz",
            destinations: .iOS,
            product: .app,
            bundleId: "io.tuist.Pinz",
            deploymentTargets: .iOS("18.0"),
            infoPlist: .extendingDefault(
                with: [
                    "UILaunchScreen": [
                        "UIColorName": "",
                        "UIImageName": "",
                    ],
                    "CFBundleURLTypes": [
                        [
                            "CFBundleURLName": "io.tuist.Pinz.invite",
                            "CFBundleURLSchemes": ["pinz"],
                            "CFBundleTypeRole": "Editor",
                        ],
                    ],
                ]
            ),
            sources: ["Pinz/Sources/**"],
            resources: ["Pinz/Resources/**"],
            entitlements: .dictionary([
                "com.apple.developer.associated-domains": .array([
                    .string("webcredentials:pinz.website"),
                    .string("applinks:pinz.website"),
                ]),
                "keychain-access-groups": .array([
                    .string("$(AppIdentifierPrefix)io.tuist.Pinz")
                ])
            ]),
            dependencies: [
                .project(target: "PinzBase", path: "Modules/PinzBase"),
                .project(target: "PinzDomain", path: "Modules/PinzDomain"),
                .project(target: "PinzAuthentication", path: "Modules/PinzAuthentication"),
                .project(target: "PinzProfile", path: "Modules/PinzProfile"),
                .project(target: "PinzNetworking", path: "Modules/PinzNetworking"),
                .project(target: "PinzAccessibility", path: "Modules/PinzAccessibility"),
                .project(target: "PinzUI", path: "Modules/PinzUI"),
                .project(target: "PinzNavigation", path: "Modules/PinzNavigation"),
                .project(target: "PinzTrips", path: "Modules/PinzTrips"),
                .project(target: "PinzPins", path: "Modules/PinzPins"),
                .project(target: "PinzFeed", path: "Modules/PinzFeed"),
                .project(target: "PinzMedias", path: "Modules/PinzMedias")
            ],
            settings: .settings(
                base: [
                    "ASSETCATALOG_COMPILER_APPICON_NAME": "PinzLightP",
                    "ASSETCATALOG_COMPILER_INCLUDE_ALL_APPICON_ASSETS": "YES",
                    "DEVELOPMENT_TEAM": "ABNY5S6RA5",
                    "CODE_SIGN_STYLE": "Automatic",
                    "CODE_SIGN_ALLOW_ENTITLEMENTS_MODIFICATION": "YES",
                ]
            )
        ),
        .target(
            name: "PinzTests",
            destinations: .iOS,
            product: .unitTests,
            bundleId: "io.tuist.PinzTests",
            deploymentTargets: .iOS("18.0"),
            infoPlist: .default,
            sources: ["Pinz/Tests/**"],
            resources: [],
            dependencies: [.target(name: "Pinz")]
        ),
        .target(
            name: "PinzUITests",
            destinations: .iOS,
            product: .uiTests,
            bundleId: "io.tuist.PinzUITests",
            deploymentTargets: .iOS("18.0"),
            infoPlist: .default,
            sources: ["Pinz/UITests/**"],
            resources: [],
            dependencies: [
                .target(name: "Pinz"),
                .external(name: "Vapor"),
                .project(target: "PinzBase", path: "Modules/PinzBase"),
                .project(target: "PinzAccessibility", path: "Modules/PinzAccessibility")
            ]
        )
    ],
    schemes: [
        .scheme(
            name: "Pinz",
            shared: true,
            buildAction: .buildAction(targets: [.target("Pinz")]),
            testAction: .testPlans([
                .relativeToManifest("PinzUnitTests.xctestplan"),
                .relativeToManifest("PinzUITests.xctestplan")
            ]),
            runAction: .runAction(executable: .target("Pinz"))
        )
    ]
)
