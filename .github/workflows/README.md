# GitHub Actions Workflows

Эта директория содержит GitHub Actions workflows для автоматической сборки и публикации Lionheart.

## Workflows

### 1. build.yml — Основная сборка

**Запуск:** вручную (workflow_dispatch)

**Параметры:**
- `version` — версия релиза (например, `v1.4.0`)
- `release` — создать релиз на GitHub
- `build_cli` — собрать CLI бинарники
- `build_android` — собрать Android AAR и APK

**Собирает:**
- CLI бинарники для Linux (amd64, arm64, 386)
- CLI бинарники для macOS (amd64, arm64)
- CLI бинарники для Windows (amd64, arm64)
- CLI бинарники для FreeBSD (amd64)
- Android AAR библиотеку
- Android APK (debug и release)

**Пример использования:**
```bash
# Через GitHub CLI
gh workflow run build.yml \
  -f version=v1.4.0 \
  -f release=true \
  -f build_cli=true \
  -f build_android=true
```

### 2. ci.yml — Непрерывная интеграция

**Запуск:** автоматически при пуше в `main`/`develop` и при PR

**Проверяет:**
- Линтинг (golangci-lint)
- Форматирование кода
- Сборку всех компонентов
- Юнит-тесты
- Кросс-компиляцию
- Сборку Android
- Сканирование безопасности (gosec)
- Проверку лицензий

### 3. nightly.yml — Ночная сборка

**Запуск:**
- Автоматически каждый день в 00:00 UTC
- Вручную (workflow_dispatch)

**Создаёт:**
- Ночную сборку CLI бинарников
- Обновляет тег `nightly` с новыми артефактами

### 4. docker.yml — Сборка Docker образов

**Запуск:**
- Автоматически при пуше тегов `v*`
- Вручную (workflow_dispatch)

**Собирает:**
- Docker образ для клиента
- Docker образ для сервера
- Поддерживает платформы: linux/amd64, linux/arm64

**Публикация:**
```bash
# Вручную с публикацией
gh workflow run docker.yml -f version=v1.4.0 -f push=true
```

## Настройка

### Требования

1. **Go 1.22** — указан в `env.GO_VERSION`
2. **Java 17** — для Android сборки
3. **Android SDK** — автоматически устанавливается
4. **NDK 25.2.9519653** — для Android сборки

### Секреты

Для публикации Docker образов требуется:
- `GITHUB_TOKEN` — автоматически предоставляется GitHub Actions

### Разрешения

Для создания релизов workflow требует:
```yaml
permissions:
  contents: write
  packages: write
```

## Артефакты

Все workflow сохраняют артефакты, которые можно скачать:
1. Перейдите в **Actions**
2. Выберите запуск workflow
3. Прокрутите вниз до раздела **Artifacts**

## Устранение неполадок

### Сборка Android завершается с ошибкой

1. Проверьте, что `go.mod` содержит правильные зависимости
2. Убедитесь, что `gomobile` установлен
3. Проверьте путь к NDK

### Docker сборка не публикуется

1. Проверьте, что `push=true` указан в параметрах
2. Убедитесь, что у workflow есть разрешение `packages: write`

### Релиз не создаётся

1. Проверьте, что `release=true` указан в параметрах
2. Убедитесь, что у workflow есть разрешение `contents: write`
3. Проверьте, что версия не конфликтует с существующим тегом
