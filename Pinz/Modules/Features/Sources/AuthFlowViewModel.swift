import Foundation
import SwiftUI
import MapKit
import Base

@Observable
public class AuthFlowViewModel {

    private enum Constants {
        static let initialCameraDistance: Double = 40000000
        static let zoomedCameraDistance: Double = 28000000
        static let cameraRotationUpdateInterval: Double = 0.016
    }

    public enum State: Equatable {
        case welcome
        case emailInput
    }

    public enum Intent {
        case startRotation
        case updateRotation
        case zoomCamera(to: Double, duration: Double, completion: (() -> Void)?)
        case performZoomAnimation(targetDistance: Double, startDistance: Double, startTime: Date, completion: (() -> Void)?)

        case proceedFromWelcome
        case back
    }
    
    public var state: State = .welcome
    public var longitude: Double = 0
    public var cameraDistance: Double = Constants.initialCameraDistance
    public var isZoomedIn: Bool = false
    public var email: String = ""
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
    
    public init() {}
    
    public func dispatch(_ intent: Intent) {
        switch intent {
        case .startRotation:
            handleStartRotation()
        case .updateRotation:
            handleUpdateRotation()
        case .zoomCamera(let targetDistance, let duration, let completion):
            handleZoomCamera(to: targetDistance, duration: duration, completion: completion)
        case .performZoomAnimation(let targetDistance, let startDistance, let startTime, let completion):
            handleZoomAnimation(
                targetDistance: targetDistance,
                startDistance: startDistance,
                startTime: startTime,
                completion: completion
            )
        case .proceedFromWelcome:
            handleProceedFromWelcome()
        case .back:
            handleBack()
        }
    }
    
    private func handleStartRotation() {
        rotationTimer?.invalidate()
        rotationTimer = Timer.scheduledTimer(
            withTimeInterval: Constants.cameraRotationUpdateInterval,
            repeats: true
        ) { [weak self] _ in
            self?.dispatch(.updateRotation)
        }
    }
    
    private func handleUpdateRotation() {
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
    
    private func handleZoomCamera(to targetDistance: Double, duration: Double, completion: (() -> Void)?) {
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
    
    private func handleZoomAnimation(targetDistance: Double, startDistance: Double, startTime: Date, completion: (() -> Void)?) {
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
    
    private func handleProceedFromWelcome() {
        isZoomedIn = true
        
        dispatch(.zoomCamera(to: Constants.zoomedCameraDistance, duration: 1.5) { [weak self] in
            self?.state = .emailInput
        })
    }
    
    private func handleBack() {
        switch state {
        case .welcome:
            break
            
        case .emailInput:
            isZoomedIn = false
            state = .welcome
            dispatch(.zoomCamera(to: Constants.initialCameraDistance, duration: 1.5, completion: nil))
        }
    }
    
    deinit {
        rotationTimer?.invalidate()
        zoomTimer?.invalidate()
    }
}

