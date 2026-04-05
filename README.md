# Lionheart v1.4 — sing-box Integration

**Lionheart VPN с интеграцией sing-box** — приватный децентрализованный self-hosted туннель с поддержкой продвинутых правил маршрутизации.

## Новое в v1.4

- **sing-box ядро** — современное ядро с TUN интерфейсом
- **Правила маршрутизации** — GeoIP, GeoSite, домены, IP, порты
- **Предустановленные профили** — готовые конфигурации для разных сценариев
- **DNS routing** — продвинутая маршрутизация DNS-запросов
- **Ad blocking** — блокировка рекламы и трекеров
- **Обратная совместимость** — сохранена поддержка legacy KCP

## Быстрый старт

### CLI

```bash
# Сервер
./lionheart
# Выберите режим 1

# Клиент с sing-box
./lionheart
# Выберите режим 3
# Введите Smart Key
# Выберите профиль маршрутизации
```

### Android

```kotlin
val vpn = LionheartVPN.getInstance()
vpn.configure(smartKey)
vpn.enableSingBox(true)
vpn.setRoutingPreset("russia")
vpn.connect()
```

## Профили маршрутизации

| Профиль | Описание |
|---------|----------|
| `direct_cn` | Прямое соединение для китайских сайтов |
| `adblock` | Блокировка рекламы и трекеров |
| `streaming` | Оптимизация для стриминговых сервисов |
| `gaming` | Низкая задержка для игр |
| `privacy` | Весь трафик через прокси |
| `russia` | Оптимизация для российских пользователей |
| `belarus` | Оптимизация для белорусских пользователей |
| `minimal` | Только блокировка рекламы |

## Структура проекта

```
lionheart-singbox/
├── core/              # Ядро с sing-box интеграцией
├── cmd/lionheart/     # CLI клиент
├── mobile/android/    # Android библиотека
├── config/examples/   # Примеры конфигураций
└── docs/              # Документация
```

## Сборка

### Требования

- Go 1.22+
- Android SDK (для Android)
- NDK (для Android)

### CLI

```bash
cd cmd/lionheart
go build -o lionheart
```

### Android

```bash
cd mobile/android/golib
gomobile bind -target=android -o ../app/libs/liblionheart.aar
```

## Конфигурация

### Пример конфигурации (config.json)

```json
{
  "role": "client",
  "client_peer": "1.2.3.4:8443",
  "password": "...",
  "use_singbox": true,
  "routing_preset": "russia"
}
```

## Сборка через GitHub Actions

### Ручная сборка (workflow_dispatch)

1. Перейдите в раздел **Actions** → **Build Lionheart**
2. Нажмите **Run workflow**
3. Укажите параметры:
   - **Version**: версия (например, `v1.4.0`)
   - **Create release**: создать релиз на GitHub
   - **Build CLI binaries**: собрать CLI бинарники
   - **Build Android AAR**: собрать Android библиотеку

### Автоматическая сборка

- **CI**: проверка кода при пуше и PR
- **Nightly**: ежедневная сборка в 00:00 UTC
- **Docker**: сборка Docker образов при пуше тегов

### Docker

```bash
# Сервер
docker run -d \
  -e LIONHEART_PASSWORD=your_password \
  -p 8443:8443/udp \
  -p 8443:8443/tcp \
  ghcr.io/yourusername/lionheart-server:latest
```

## Документация

- [Миграция на sing-box](docs/SINGBOX_MIGRATION.md)
- [Примеры конфигураций](config/examples/)
- [GitHub Actions](.github/workflows/)

## Лицензия

MIT License
