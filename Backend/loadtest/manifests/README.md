# Стабы внешних зависимостей (loadtest)

Применять только в namespace `pinz-loadtest`. На прод не выкатывать.

## Что чем подменяется

| Внешняя зависимость | Чем заменяется в контуре | Как |
|---|---|---|
| Yandex Object Storage (S3) | **MinIO** | `minio.yaml` поднимает MinIO + Job создаёт bucket `pinz-loadtest`. Backend использует через env `S3_ENDPOINT=http://minio.pinz-loadtest.svc.cluster.local:9000`, `S3_KEY_PREFIX=loadtest/`. |
| Maileroo SMTP | **Mailpit** | `mailpit.yaml`. Backend через env `SMTP_HOST=mailpit.pinz-loadtest.svc.cluster.local`, `SMTP_PORT=1025`. UI на 8025. |
| BigDataCloud reverse-geocoding | Самописный **geo-stub** | `geo-stub/` — Go HTTP. Образ: `docker build -t pinz/geo-stub:loadtest geo-stub/` + `kubectl apply -f geo-stub/k8s.yaml`. Backend через env `GEOCODING_BASE_URL=http://geo-stub.pinz-loadtest.svc.cluster.local:8080`. |
| APNS | **Не подменяем** — отключаем sender | Достаточно не задавать `APNS_KEY_BASE64`. В `notification-service/internal/apns/sender.go` это автоматически выключает push-доставку. |
| ML-модуль (Python) | **Не подменяем** — обработка встроенная в trip-service worker | Stream `pinz:trip:ml:tasks` читает `trip-service/internal/worker`, ML-задачи обрабатываются in-process. Реальной внешней ML-зависимости в текущей сборке нет. |

## Применение

```bash
# из Backend/
kubectl apply -f loadtest/manifests/minio.yaml
kubectl apply -f loadtest/manifests/mailpit.yaml

# geo-stub нужно сначала собрать локально
docker build -t pinz/geo-stub:loadtest loadtest/manifests/geo-stub/
# в minikube образ загружается через `minikube image load`, в k3s — через imageImport
kubectl apply -f loadtest/manifests/geo-stub/k8s.yaml
```

## Очистка

```bash
kubectl delete namespace pinz-loadtest
```
