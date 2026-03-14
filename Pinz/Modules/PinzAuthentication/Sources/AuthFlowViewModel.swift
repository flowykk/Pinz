import Foundation
import SwiftUI
import MapKit
import PinzBase
import PinzNetworking

@MainActor @Observable
final class AuthFlowViewModel {

    private enum Constants {
        static let initialCameraDistance: Double = 40000000
        static let zoomedCameraDistance: Double = 28000000
        static let cameraRotationUpdateInterval: Double = 0.016
    }

    enum State: Equatable {
        case welcome
        case email
        case login(LoginState)
        case register(RegisterState)
    }

    enum LoginState: Equatable {
        case passkeyPrompt
    }

    enum RegisterState: Equatable {
        case code
        case nickname
        case passkeyPrompt
    }

    enum Intent {
        case startRotation
        case updateRotation
        case zoomCamera(to: Double, duration: Double, completion: (() -> Void)?)
        case performZoomAnimation(targetDistance: Double, startDistance: Double, startTime: Date, completion: (() -> Void)?)

        case proceedFromWelcome
        case back
    }

    enum AsyncIntent {
        case `continue`
    }

    var state: State = .welcome
    var longitude: Double = 0
    var cameraDistance: Double = Constants.initialCameraDistance
    var isZoomedIn: Bool = false
    var text: String = ""
    var cameraPosition: MapCameraPosition = .camera(
        MapCamera(
            centerCoordinate: CLLocationCoordinate2D(latitude: 0, longitude: 0),
            distance: Constants.initialCameraDistance,
            heading: 0,
            pitch: 0
        )
    )

    nonisolated(unsafe) private var rotationTimer: Timer?
    nonisolated(unsafe) private var zoomTimer: Timer?

    private var email: String = ""
    private var registrationId: String = ""
    private var username: String = ""

    private let networkService = NetworkService()
    private let passkeyService = PasskeyService()
    private var router: AppRouting?

    init() {}

    func setRouter(_ router: AppRouting?) {
        self.router = router
    }

    func dispatch(_ intent: Intent) {
        switch intent {
        case .startRotation:
            startRotation()
        case .updateRotation:
            updateRotation()
        case .zoomCamera(let targetDistance, let duration, let completion):
            zoomCamera(to: targetDistance, duration: duration, completion: completion)
        case .performZoomAnimation(let targetDistance, let startDistance, let startTime, let completion):
            performZoomAnimation(
                targetDistance: targetDistance,
                startDistance: startDistance,
                startTime: startTime,
                completion: completion
            )
        case .proceedFromWelcome:
            proceedFromWelcome()
        case .back:
            back()
        }
    }

    func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .continue:
            switch state {
            case .email:
                try await proceedFromEmail()
            case let .register(registerState):
                switch registerState {
                case .code:
                    try await proceedFromRegisterCode()
                case .nickname:
                    try await proceedFromRegisterNickname()
                case .passkeyPrompt:
                    break
                }
            default:
                break
            }
        }
        text = ""
    }

    private func startRotation() {
        rotationTimer?.invalidate()
        rotationTimer = Timer.scheduledTimer(
            withTimeInterval: Constants.cameraRotationUpdateInterval,
            repeats: true
        ) { [weak self] _ in
            self?.dispatch(.updateRotation)
        }
    }

    private func updateRotation() {
        longitude += 0.01
        if longitude > 180 {
            longitude = -180
        }

        cameraPosition = .camera(
            MapCamera(
                centerCoordinate: CLLocationCoordinate2D(latitude: 0, longitude: longitude),
                distance: cameraDistance,
                heading: 0,
                pitch: 0
            )
        )
    }

    private func zoomCamera(to targetDistance: Double, duration: Double, completion: (() -> Void)?) {
        let startDistance = cameraDistance
        let startTime = Date()

        zoomTimer?.invalidate()
        zoomTimer = Timer.scheduledTimer(
            withTimeInterval: Constants.cameraRotationUpdateInterval,
            repeats: true
        ) { [weak self] _ in
            self?.dispatch(.performZoomAnimation(
                targetDistance: targetDistance,
                startDistance: startDistance,
                startTime: startTime,
                completion: completion
            ))
        }
    }

    private func performZoomAnimation(targetDistance: Double, startDistance: Double, startTime: Date, completion: (() -> Void)?) {
        let duration: Double = 2.0
        let elapsed = Date().timeIntervalSince(startTime)
        let progress = min(elapsed / duration, 1.0)

        let easedProgress = HelperFunctions.easeInOutCubic(progress)
        cameraDistance = startDistance + (targetDistance - startDistance) * easedProgress

        if progress >= 1.0 {
            zoomTimer?.invalidate()
            zoomTimer = nil
            cameraDistance = targetDistance
            completion?()
        }
    }

    private func proceedFromWelcome() {
        isZoomedIn = true

        dispatch(.zoomCamera(to: Constants.zoomedCameraDistance, duration: 1.5) { [weak self] in
            self?.state = .email
        })
    }

    private func proceedFromEmail() async throws {
        let response = try await networkService.submitEmail(email: text)
        email = text
        if response.isRegistered {
            state = .login(.passkeyPrompt)
            try await performLogin()
        } else {
            registrationId = response.registrationId ?? ""
            state = .register(.code)
        }
    }

    private func proceedFromRegisterCode() async throws {
        try await networkService.verifyEmail(
            registrationId: registrationId,
            verificationCode: text
        )
        state = .register(.nickname)
    }

    private func proceedFromRegisterNickname() async throws {
        username = text
        state = .register(.passkeyPrompt)
        do {
            try await performRegistration()
        } catch {
            print("[AuthFlow] Registration failed: \(error)")
            state = .register(.nickname)
            throw error
        }
    }

    private func performLogin() async throws {
        do {
            let options = try await networkService.passkeyLoginBegin(email: email)
            let credentialJSON = try await passkeyService.performAssertion(optionsJSON: options.optionsJson)
            let tokens = try await networkService.passkeyLoginFinish(email: email, credentialJSON: credentialJSON)
            TokenStorage.shared.save(accessToken: tokens.accessToken, refreshToken: tokens.refreshToken)
            router?.navigateToMain()
        } catch {
            print("[AuthFlow] Login failed: \(error)")
            state = .login(.passkeyPrompt)
            throw error
        }
    }

    private func performRegistration() async throws {
        let options = try await networkService.passkeyRegisterBegin(
            registrationId: registrationId,
            username: username
        )
        let credentialJSON = try await passkeyService.performAttestation(optionsJSON: options.optionsJson)
        let tokens = try await networkService.passkeyRegisterFinish(
            registrationId: registrationId,
            credentialJSON: credentialJSON
        )
        TokenStorage.shared.save(accessToken: tokens.accessToken, refreshToken: tokens.refreshToken)
        router?.navigateToMain()
    }

    private func back() {
        switch state {
        case .email:
            isZoomedIn = false
            state = .welcome
            dispatch(.zoomCamera(to: Constants.initialCameraDistance, duration: 1.5, completion: nil))
        case .login:
            state = .email
        case let .register(registerState):
            switch registerState {
            case .code:
                state = .email
            case .nickname:
                state = .register(.code)
            case .passkeyPrompt:
                state = .register(.nickname)
            }
        default:
            break
        }
    }

    deinit {
        rotationTimer?.invalidate()
        zoomTimer?.invalidate()
    }
}
