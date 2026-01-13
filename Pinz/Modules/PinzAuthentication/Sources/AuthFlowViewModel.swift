import Foundation
import SwiftUI
import MapKit
import PinzBase
import PinzNetworking

@Observable
public class AuthFlowViewModel {

    private enum Constants {
        static let initialCameraDistance: Double = 40000000
        static let zoomedCameraDistance: Double = 28000000
        static let cameraRotationUpdateInterval: Double = 0.016
    }

    public enum State: Equatable {
        case welcome
        case email
        case auth(AuthState)
        case register(RegisterState)
    }

    public enum AuthState: Equatable {
        case password
    }

    public enum RegisterState: Equatable {
        case code
        case password
        case repeatPassword
        case nickname
    }

    public enum Intent {
        case startRotation
        case updateRotation
        case zoomCamera(to: Double, duration: Double, completion: (() -> Void)?)
        case performZoomAnimation(targetDistance: Double, startDistance: Double, startTime: Date, completion: (() -> Void)?)

        case proceedFromWelcome
        case back
    }

    public enum AsyncIntent {
        case `continue`
    }

    public var state: State = .welcome
    public var longitude: Double = 0
    public var cameraDistance: Double = Constants.initialCameraDistance
    public var isZoomedIn: Bool = false
    public var text: String = ""
    public var cameraPosition: MapCameraPosition = .camera(
        MapCamera(
            centerCoordinate: CLLocationCoordinate2D(latitude: 0, longitude: 0),
            distance: Constants.initialCameraDistance,
            heading: 0,
            pitch: 0
        )
    )
    
    private var rotationTimer: Timer?
    private var zoomTimer: Timer?

    private let networkService = NetworkService()

    public init() {}
    
    public func dispatch(_ intent: Intent) {
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

    public func asyncDispatch(_ intent: AsyncIntent) async throws {
        switch intent {
        case .continue:
            switch state {
            case .email:
                try await proceedFromEmail()
            case .auth:
                try await proceedFromAuthPassword()
            case let .register(registerState):
                switch registerState {
                case .code:
                    try await proceedFromRegisterCode()
                case .password:
                    try await proceedFromRegisterPassword()
                case .repeatPassword:
                    try await proceedFromRegisterRepeatPassword()
                case .nickname:
                    try await proceedFromRegisterNickname()
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
        let userExists = try await networkService.checkEmail(email: text)
        if userExists.success {
            state = .auth(.password)
        } else {
            state = .register(.code)
        }
        text = ""
    }

    private func proceedFromAuthPassword() async throws {
        print("Gone to Feed")
    }

    private func proceedFromRegisterCode() async throws {
        state = .register(.password)
    }

    private func proceedFromRegisterPassword() async throws {
        state = .register(.repeatPassword)
    }

    private func proceedFromRegisterRepeatPassword() async throws {
        state = .register(.nickname)
    }

    private func proceedFromRegisterNickname() async throws {
        print("Gone to Feed")
    }

    private func back() {
        switch state {
        case .email:
            isZoomedIn = false
            state = .welcome
            dispatch(.zoomCamera(to: Constants.initialCameraDistance, duration: 1.5, completion: nil))
        case .auth:
            state = .email
        case let .register(registerState):
            switch registerState {
            case .code, .password:
                state = .email
            case .repeatPassword:
                state = .register(.password)
            case .nickname:
                state = .register(.repeatPassword)
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

