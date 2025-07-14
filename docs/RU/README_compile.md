# **Детальное руководство по сборке программы "Prepare After Updater"**

## **1. Требования к системе**
Перед сборкой необходимо установить:
1. **Go 1.16+** (рекомендуется последняя стабильная версия).
   - Проверить установку:  
     ```bash
     go version
     ```
   - Если не установлен:  
     ```bash
     sudo apt install golang  # Для Debian/Ubuntu
     sudo yum install golang  # Для CentOS/RHEL
     ```
2. **Git** (для управления версиями и зависимостями).
   ```bash
   sudo apt install git
   ```

---

## **2. Подготовка рабочего окружения**
### **2.1. Клонирование репозитория**
Если исходный код хранится в Git:
```bash
git clone https://github.com/cavblue/prepare-after-updater.git
cd prepare-after-updater
```

### **2.2. Инициализация Go-модуля**
Если проект не использует `go.mod`:
```bash
go mod init github.com/cavblue/prepare-after-updater
```
Это создаст файл `go.mod` для управления зависимостями.

---

## **3. Установка зависимостей**
Программа использует только стандартные библиотеки Go (`encoding/json`, `os/exec`, `net/http` и др.), поэтому явные зависимости не требуются. Однако, если есть внешние пакеты:
```bash
go mod tidy
```
Эта команда:
- Скачает недостающие зависимости.
- Удалит неиспользуемые.

---

## **4. Сборка программы**
### **4.1. Базовая сборка (для текущей ОС)**
```bash
go build -o prepare-after-updater /cmd/app/main.go
```
- Флаг `-o` задает имя выходного файла (`prepare-after-updater`).
- Исполняемый файл появится в текущей директории.

### **4.2. Кросс-платформенная сборка**
Go поддерживает сборку под разные ОС и архитектуры. Примеры:

#### **Для Linux (64-bit)**
```bash
GOOS=linux GOARCH=amd64 go build -o prepare-after-updater-linux-amd64 /cmd/app/main.go
```

<div hidden>
#### **macOS (ARM/M1)**
```bash
GOOS=darwin GOARCH=arm64 go build -o prepare-after-updater-macos-arm64
```
</div>

#### **Полный список поддерживаемых платформ**
```bash
go tool dist list
```

---

## **5. Тестирование сборки**
### **5.1. Проверка запуска**
```bash
./prepare-after-updater --help
```
Должна отобразиться справка по использованию.

### **5.2. Запуск с тестовым конфигом**
```bash
./prepare-after-updater --autoconfig test-config.json
```
- Проверяет генерацию шаблона конфигурации.

### **5.3. Тестирование установки**
```bash
sudo ./prepare-after-updater --user $(whoami) --download file://$(pwd)/test-config.json
```
- Запускает обработку для текущего пользователя с локальным конфигом.

---

## **6. Установка программы в систему**
### **6.1. Копирование бинарного файла**
```bash
sudo cp prepare-after-updater /usr/local/bin/
```
Теперь программа доступна из любого места:
```bash
prepare-after-updater --version
```

### **6.2. Создание системных unit-файлов (для systemd)**
Если программа должна запускаться автоматически, создайте файл `/etc/systemd/system/prepare-after-updater.service`:
```ini
[Unit]
Description=Prepare After Updater
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/prepare-after-updater --config /etc/prepare-after-updater/config.json
Restart=on-failure

[Install]
WantedBy=multi-user.target
```
Затем:
```bash
sudo systemctl daemon-reload
sudo systemctl enable prepare-after-updater
sudo systemctl start prepare-after-updater
```

---

## **7. Создание DEB/RPM пакетов (опционально)**
### **7.1. Для Debian/Ubuntu (DEB)**
1. Установите `dpkg-deb`:
   ```bash
   sudo apt install dpkg-dev
   ```
2. Создайте структуру пакета:
   ```bash
   mkdir -p deb-package/usr/local/bin
   mkdir -p deb-package/etc/prepare-after-updater
   cp prepare-after-updater deb-package/usr/local/bin/
   cp config.json deb-package/etc/prepare-after-updater/
   ```
3. Создайте файл `deb-package/DEBIAN/control`:
   ```plaintext
   Package: prepare-after-updater
   Version: 2.1.4
   Section: utils
   Priority: optional
   Architecture: amd64
   Maintainer: Ваше Имя <your@email.com>
   Description: Программа для настройки окружения после обновления.
   ```
4. Соберите пакет:
   ```bash
   dpkg-deb --build deb-package prepare-after-updater_2.1.4_amd64.deb
   ```

### **7.2. Для CentOS/RHEL (RPM)**
1. Установите `rpm-build`:
   ```bash
   sudo yum install rpm-build
   ```
2. Создайте структуру пакета:
   ```bash
   mkdir -p rpm-package/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
   ```
3. Создайте файл `rpm-package/SPECS/prepare-after-updater.spec`:
   ```plaintext
   Name: prepare-after-updater
   Version: 2.1.4
   Release: 1
   Summary: Программа для настройки окружения после обновления.
   License: MIT
   URL: https://github.com/ваш-репозиторий/prepare-after-updater
   Source0: prepare-after-updater
   BuildArch: x86_64

   %description
   Программа для автоматической настройки пользовательских окружений после обновления системы.

   %install
   mkdir -p %{buildroot}/usr/local/bin
   mkdir -p %{buildroot}/etc/prepare-after-updater
   install -m 755 %{SOURCE0} %{buildroot}/usr/local/bin/
   install -m 644 config.json %{buildroot}/etc/prepare-after-updater/

   %files
   /usr/local/bin/prepare-after-updater
   /etc/prepare-after-updater/config.json
   ```
4. Соберите пакет:
   ```bash
   rpmbuild -bb rpm-package/SPECS/prepare-after-updater.spec
   ```

---

## **8. Проверка целостности сборки**
### **8.1. Тестирование на чистой системе**
1. Разверните виртуальную машину с чистым дистрибутивом (Ubuntu/CentOS).
2. Установите пакет:
   ```bash
   sudo dpkg -i prepare-after-updater_2.1.4_amd64.deb  # Для Debian
   sudo rpm -ivh prepare-after-updater-2.1.4-1.x86_64.rpm  # Для CentOS
   ```
3. Проверьте работу:
   ```bash
   sudo prepare-after-updater --version
   ```

---

## **9. Деплой в production**
1. Загрузите пакет в репозиторий (например, S3 или локальный Artifactory).
2. Настройте автоматическую установку через Ansible/Puppet:
   ```yaml
   # Пример для Ansible
   - name: Install prepare-after-updater
     apt:
       deb: "https://your-repo/prepare-after-updater_2.1.4_amd64.deb"
   ```

---

## **10. Возможные проблемы и решения**
| Проблема | Решение |
|----------|---------|
| `go: cannot find main module` | Выполните `go mod init` |
| Нет прав на запись в `/usr/local/bin` | Используйте `sudo` |
| Ошибка `undefined: http.Get` | Проверьте `GO111MODULE=on` |
| Программа не запускается после установки | Проверьте `ldd` (для Linux) или права `chmod +x` |

---

### **Итог**
Программа собирается в **3 шага**:
1. `go build` — сборка бинарника.
2. `go test` — проверка работоспособности.
3. `dpkg-deb`/`rpmbuild` — упаковка (опционально).

Для кросс-платформенной поддержки используйте `GOOS` и `GOARCH`. Готовые пакеты можно распространять через DEB/RPM-репозитории.