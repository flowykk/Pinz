# Развертывание Pinz Backend

## Быстрый старт

### Production сервер (рекомендуемый способ)

```bash
# Полная автоматизированная настройка сервера
wget https://raw.githubusercontent.com/flowykk/Pinz/main/Backend/setup-server.sh
chmod +x setup-server.sh
./setup-server.sh --repo-url https://github.com/flowykk/Pinz.git
```

Скрипт автоматически:
1. Установит все компоненты (Docker, k3s, Helm, Istio)
2. Склонирует репозиторий
3. Настроит инфраструктуру
4. Создаст переменные окружения
5. Подготовит CI/CD доступ
6. **Сделает make команды работоспособными**

### После настройки сервера

```bash
cd /opt/pinz/Backend

# Все make команды теперь работают
make infra-up      # Запуск инфраструктуры
make k8s-deploy    # Деплой приложения

# Или универсальный деплой
./deploy.sh --environment prod
```

## Универсальный скрипт deploy.sh

Скрипт `deploy.sh` работает как локально, так и в CI/CD:

```bash
# Деплой с последней версией
./deploy.sh

# Деплой с конкретным тегом
./deploy.sh --image-tag v1.2.3

# Через переменные окружения
IMAGE_TAG=v1.2.3 ./deploy.sh
```

### Переменные окружения

| Переменная | Описание | Значение по умолчанию |
|------------|----------|----------------------|
| `IMAGE_TAG` | Тег Docker образа | latest |
| `DOCKER_REGISTRY` | Registry для образов | ghcr.io |
| `DOCKER_REPO` | Репозиторий образов | owner/repo |
| `POSTGRES_PASSWORD` | Пароль БД | pinz_password |
| `JWT_SECRET_KEY` | JWT секрет | change-me-in-production |

## CI/CD настройка

### GitHub Actions

Проект использует автоматическое развертывание:

- **main ветка** → production окружение
- **develop ветка** → staging окружение

### Необходимые GitHub Secrets

В `Settings → Secrets and variables → Actions` добавьте:

| Secret | Значение | Пример |
|--------|----------|--------|
| `VPS_HOST` | IP сервера | `192.168.1.100` |
| `VPS_USER` | SSH пользователь | `deploy` |
| `VPS_SSH_KEY` | Приватный SSH ключ | `-----BEGIN OPENSSH PRIVATE KEY-----...` |
| `POSTGRES_PASSWORD` | Пароль PostgreSQL | `my_secure_db_password` |
| `JWT_SECRET_KEY` | JWT секретный ключ | `my_jwt_secret_key` |

### Генерация SSH ключа

```bash
# Создание ключа
ssh-keygen -t ed25519 -C "github-cd@your-repo" -f ~/.ssh/pinz-deploy

# Копирование на сервер
ssh-copy-id -i ~/.ssh/pinz-deploy.pub user@your-server

# Добавление приватного ключа в GitHub secret
cat ~/.ssh/pinz-deploy
```

## Мониторинг развертывания

### Статус подов

```bash
kubectl get pods
kubectl get deployments
kubectl get services
```

### Логи

```bash
# Логи приложений
kubectl logs -f deployment/api-gateway
kubectl logs -f deployment/auth-service

# Логи Istio
kubectl logs -n istio-system deployment/istiod
```

### Ресурсы

```bash
# Использование CPU/памяти
kubectl top nodes
kubectl top pods

# Детали подов
kubectl describe pod <pod-name>
```

### Health check

```bash
# API health
curl http://localhost:8080/health

# Swagger
curl http://localhost:8080/swagger/index.html
```

## Troubleshooting

### Проблема: Образы не скачиваются

```bash
# Проверка аутентификации
docker login ghcr.io

# Ручная загрузка
docker pull ghcr.io/owner/repo/pinz-api-gateway:latest
```

### Проблема: Деплой не стартует

```bash
# Проверка статуса
kubectl get pods
kubectl describe deployment api-gateway

# Логи
kubectl logs -f deployment/api-gateway
```

### Проблема: Сервис недоступен

```bash
# Проверка сервисов
kubectl get services
kubectl describe service api-gateway

# Проверка Istio
istioctl proxy-status
```

## Rollback

### Откат к предыдущей версии

```bash
# Просмотр истории
kubectl rollout history deployment/api-gateway

# Откат
kubectl rollout undo deployment/api-gateway
kubectl rollout undo deployment/auth-service
```

### Ручной откат образа

```bash
# Изменение тега образа
kubectl set image deployment/api-gateway api-gateway=ghcr.io/owner/repo/pinz-api-gateway:v1.1.0
kubectl set image deployment/auth-service auth-service=ghcr.io/owner/repo/pinz-auth-service:v1.1.0

# Перезапуск
kubectl rollout restart deployment/api-gateway deployment/auth-service
```

## Масштабирование

```bash
# Изменение количества реплик
kubectl scale deployment api-gateway --replicas=3
kubectl scale deployment auth-service --replicas=2

# Автомасштабирование (если настроено)
kubectl get hpa
```

## Безопасность

1. **Регулярно обновляйте** сервер и компоненты
2. **Мониторьте логи** на подозрительную активность
3. **Используйте strong пароли** для БД и JWT
4. **Ограничьте SSH доступ** по IP
5. **Настройте бэкапы** базы данных

## Производительность

- **Мониторьте** CPU, память, сеть, диск
- **Оптимизируйте** размеры Docker образов
- **Кешируйте** зависимости при сборке
- **Масштабируйте** при росте нагрузки