# Pinz

iOS-приложение для управления путешествиями и пинами на карте.

## Архитектура проекта

Проект построен на модульной архитектуре с использованием Tuist. Каждый модуль — это отдельный framework с четкими зависимостями.

### Модули проекта

**Базовые модули:**
- **PinzDomain** — бизнес-модели (User, Trip, Pin, MediaTag)
- **PinzBase** — общие утилиты и протоколы
- **PinzUI** — переиспользуемые UI-компоненты
- **PinzNetworking** — сетевой слой и API

**Функциональные модули:**
- **PinzAuthentication** — авторизация и регистрация
- **PinzProfile** — профиль и настройки
- **PinzTrips** — путешествия и карта
- **PinzPins** — управление пинами
- **PinzFeed** — лента контента

**Навигация:**
- **PinzNavigation** — глобальный роутер и переходы между экранами

### Зависимости между модулями

Модуль PinzNavigation зависит от всех функциональных и базовых модулей. Функциональные модули зависят только от базовых модулей и не знают о PinzNavigation, что исключает циклические зависимости. Полный граф зависимостей можно увидеть в файле graph.png, который генерируется командой `tuist graph`.

Для избежания циклических зависимостей используется протокол AppRouting из модуля PinzBase. Класс AppRouter реализует этот протокол, а функциональные модули получают роутер через Environment в SwiftUI.

### MVVM внутри модулей

Внутри каждого функционального модуля используется паттерн MVVM. View — это SwiftUI View с декларативным UI, который взаимодействует с ViewModel через метод dispatch. ViewModel — Observable класс, содержащий бизнес-логику и состояние. Для обработки действий применяется Intent-паттерн: все действия описаны в enum Intent, а метод dispatch обрабатывает каждое намерение.

Пример: ProfileViewModel содержит Intent enum с действиями типа changeState или navigateToEmailChange. При нажатии кнопки в ProfileView вызывается dispatch с соответствующим намерением, и ViewModel обрабатывает действие, обновляя состояние или инициируя навигацию через роутер.

## Устройство навигации

Навигация в приложении построена на централизованном подходе с использованием глобального роутера. Все переходы между экранами управляются из одной точки, что упрощает отслеживание навигационных потоков и позволяет избежать дублирования логики.

### Структура Route

Все возможные экраны приложения описаны в enum Route, который использует вложенные enum для группировки связанных экранов:

```swift
public enum Route: Hashable {
    case auth
    case trip(TripRoute)
    case profile(ProfileRoute)
    case feed
    case pins
    case members
}

public enum TripRoute: Hashable {
    case info(trip: Trip)
    case profile(user: User)
}

public enum ProfileRoute: Hashable {
    case profile
    case emailChange(email: String, action: EmailChangeAction)
}
```

Такая структура позволяет легко добавлять новые экраны и группировать их по функциональным модулям.

### AppRouter

AppRouter управляет навигационным стеком и предоставляет методы для переходов:

```swift
@MainActor @Observable
public final class AppRouter: AppRouting {
    public var path: [Route] = []
    
    public func navigate(to route: Route) {
        path.append(route)
    }
    
    public func pop() {
        guard !path.isEmpty else { return }
        path.removeLast()
    }
}
```

Роутер содержит массив path, который используется в NavigationStack. Методы navigate и pop добавляют или удаляют экраны из стека.

### RootView

RootView — это корневой экран приложения, который содержит NavigationStack и отрисовывает экраны в зависимости от состояния роутера:

```swift
public struct RootView<Content: View>: View {
    @Bindable var router: AppRouter
    let rootContent: Content
    
    public var body: some View {
        NavigationStack(path: $router.path) {
            rootContent
                .navigationDestination(for: Route.self) { route in
                    destinationView(for: route)
                        .toolbar(.hidden)
                }
        }
        .environment(\.appRouter, router)
    }
    
    @ViewBuilder
    private func destinationView(for route: Route) -> some View {
        switch route {
        case .auth: AuthFlowView()
        case .profile(.profile): ProfileView(user: ...)
        case .trip(.info(let trip)): TripInfoView(trip: trip)
        // ...
        }
    }
}
```

RootView передает роутер через Environment, что позволяет любому экрану получить доступ к навигации.

### Взаимодействие с роутером

Функциональные модули не зависят от PinzNavigation напрямую. Они получают роутер через протокол AppRouting:

```swift
// В ViewModel
@Observable
class ProfileViewModel {
    private var router: AppRouting?
    
    func setRouter(_ router: AppRouting?) {
        self.router = router
    }
    
    func dispatch(_ intent: Intent) {
        switch intent {
        case .navigateToEmailChange:
            router?.navigateToEmailChange(email: user.email, action: action)
        case .back:
            router?.pop()
        }
    }
}

// В View
struct ProfileView: View {
    @State private var viewModel: ProfileViewModel
    @Environment(\.appRouter) private var router
    
    var body: some View {
        Button("Изменить email") {
            viewModel.dispatch(.navigateToEmailChange)
        }
        .onAppear { viewModel.setRouter(router) }
    }
}
```

Такой подход позволяет инициировать навигацию из ViewModel через метод dispatch, сохраняя при этом модульность архитектуры.

### Типы переходов

Поддерживаются три типа переходов:
- **Push** — стандартный переход через navigationDestination
- **Full Screen Cover** — полноэкранная модалка через .fullScreenCover
- **Sheet** — всплывающая панель снизу через .sheet с настройкой высоты через presentationDetents

## Выбор технологий

старое:
- Внедрение Tuist для модульного разделения проекта и управления сборкой приложения.
- Реализация карты нативными инструментами SwiftUI
- Реализация UI-тестов с использованием мок-сервера на Vapor
- Внедрение фреймворка Moya для реализации сетевых запросов С ЗАГЛУШКАМИ
- Реализация локализации приложения с использованием Localizable.strings в Xcode

новое:
- Выбор нативных инструментов для разработки на операционной системе ios
- Реализация работы уведомлений
- Реализация работы с Universal links
- 


