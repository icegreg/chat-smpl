# Setup Scripts для rtuccli

Данная директория содержит скрипты для автоматической настройки алиаса `rtuccli` на различных платформах.

## Доступные скрипты

### Windows PowerShell (рекомендуется)
**Файл:** `setup-rtuccli-alias.ps1`

**Запуск:**
```powershell
# Из корневой директории проекта
powershell -ExecutionPolicy Bypass -File scripts\setup-rtuccli-alias.ps1

# Или если вы уже в PowerShell
.\scripts\setup-rtuccli-alias.ps1

# Перезагрузить профиль
. $PROFILE
```

**Что делает:**
- Создает директорию профиля PowerShell (если не существует)
- Создает файл профиля `profile.ps1` (если не существует)
- Добавляет функцию `rtuccli` в профиль
- Проверяет, не существует ли функция уже

**Расположение профиля:**
`C:\Users\<username>\Documents\WindowsPowerShell\profile.ps1`

### Windows CMD
**Файл:** `setup-rtuccli-alias.bat`

**Запуск:**
```cmd
scripts\setup-rtuccli-alias.bat
```

**Что делает:**
- Создает doskey макрос `rtuccli` для текущей сессии CMD
- Показывает инструкции по постоянной настройке через реестр

**ВАЖНО:** CMD doskey макросы работают только в текущей сессии. Для постоянного алиаса используйте PowerShell.

### Linux (bash/zsh)
**Файл:** `setup-rtuccli-alias.sh`

**Запуск:**
```bash
# Сделать скрипт исполняемым (первый раз)
chmod +x scripts/setup-rtuccli-alias.sh

# Запустить скрипт
bash scripts/setup-rtuccli-alias.sh

# Перезагрузить конфигурацию
source ~/.bashrc  # для bash
source ~/.zshrc   # для zsh
```

**Что делает:**
- Определяет используемый shell (bash или zsh)
- Добавляет алиас в соответствующий конфигурационный файл
- Пытается перезагрузить конфигурацию автоматически

**Расположение конфигурации:**
- bash: `~/.bashrc`
- zsh: `~/.zshrc`

### macOS (zsh/bash)
**Файл:** `setup-rtuccli-alias.sh` (тот же что для Linux)

**Запуск:**
```bash
# macOS Catalina+ использует zsh по умолчанию
bash scripts/setup-rtuccli-alias.sh

# Перезагрузить конфигурацию
source ~/.zshrc   # для zsh (по умолчанию)
source ~/.bashrc  # для bash (старые версии)
```

## Использование rtuccli после настройки

После успешной установки алиаса вы можете использовать команду `rtuccli` из любой директории:

```bash
# Список сервисов
rtuccli service list

# Список конференций
rtuccli conf list

# Активные конференции
rtuccli conf list --status active

# Детали конференции
rtuccli conf get <conference-id>

# Участники конференции
rtuccli conf clients <conference-id>

# Получить справку
rtuccli --help
```

## Ручная настройка

Если автоматические скрипты не работают, вы можете настроить алиас вручную:

### PowerShell (вручную)
```powershell
# Открыть профиль в редакторе
notepad $PROFILE

# Добавить в конец файла:
function rtuccli {
    param(
        [Parameter(ValueFromRemainingArguments=$true)]
        [string[]]$Arguments
    )
    $cmd = $Arguments -join ' '
    docker-compose exec -T admin-service sh -c $cmd
}

# Сохранить и перезагрузить
. $PROFILE
```

### bash/zsh (вручную)
```bash
# Для bash
echo 'alias rtuccli="docker-compose exec -T admin-service sh -c"' >> ~/.bashrc
source ~/.bashrc

# Для zsh
echo 'alias rtuccli="docker-compose exec -T admin-service sh -c"' >> ~/.zshrc
source ~/.zshrc
```

## Требования

- Docker и docker-compose должны быть установлены и запущены
- admin-service должен быть запущен: `docker-compose up -d admin-service`
- Вы должны находиться в корневой директории проекта (где расположен `docker-compose.yml`)

## Troubleshooting

### PowerShell: "Execution policy restricted"
```powershell
# Запустить с bypass политики
powershell -ExecutionPolicy Bypass -File scripts\setup-rtuccli-alias.ps1
```

### Linux/macOS: "Permission denied"
```bash
# Сделать скрипт исполняемым
chmod +x scripts/setup-rtuccli-alias.sh
```

### Алиас не работает после установки
```bash
# Перезагрузить конфигурацию shell
source ~/.bashrc    # bash
source ~/.zshrc     # zsh
. $PROFILE          # PowerShell

# Или откройте новое окно терминала/PowerShell
```

### "admin-service not found"
```bash
# Убедитесь что admin-service запущен
docker-compose ps admin-service

# Запустить если не запущен
docker-compose up -d admin-service
```

## Дополнительная документация

- [RTUCCLI_GUIDE.md](../RTUCCLI_GUIDE.md) - Полное руководство по работе с rtuccli
- [services/admin/README.md](../services/admin/README.md) - Документация admin-service API
- [cmd/rtuccli/README.md](../cmd/rtuccli/README.md) - Документация CLI клиента
