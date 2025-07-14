# **In-Depth Build Guide for "Prepare After Updater"**  

The program is written in **Go (Golang)**, which enables cross-platform compilation and simple binary generation. Below is a step-by-step build process including dependency management and packaging.

---

## **1. System Requirements**
Before building, install:
1. **Go 1.16+** (latest stable version recommended):
   - Verify installation:
     ```bash
     go version
     ```
   - If not installed:
     ```bash
     sudo apt install golang  # For Debian/Ubuntu
     sudo yum install golang  # For CentOS/RHEL
     ```
2. **Git** (for version control and dependency management):
   ```bash
   sudo apt install git
   ```

---

## **2. Environment Setup**
### **2.1. Repository Cloning**
If source code is in Git:
```bash
git clone https://github.com/cavblue/prepare-after-updater.git
cd prepare-after-updater
```

### **2.2. Go Module Initialization**
If project lacks `go.mod`:
```bash
go mod init github.com/cavblue/prepare-after-updater
```
This creates `go.mod` for dependency management.

---

## **3. Dependency Installation**
The program primarily uses Go standard libraries (`encoding/json`, `os/exec`, `net/http`, etc.), so explicit dependencies aren't required. However, for external packages:
```bash
go mod tidy
```
This command:
- Downloads missing dependencies
- Removes unused packages

---

## **4. Program Compilation**
### **4.1. Basic Build (Current OS)**
```bash
go build -o prepare-after-updater /cmd/app/main.go
```
- `-o` flag specifies output filename (`prepare-after-updater`)
- Binary appears in current directory

### **4.2. Cross-Platform Compilation**
Go supports multi-OS/architecture builds. Examples:

#### **Linux (64-bit)**
```bash
GOOS=linux GOARCH=amd64 go build -o prepare-after-updater-linux-amd64 /cmd/app/main.go
```
<div hidden>
#### **macOS (ARM/M1)**
```bash
GOOS=darwin GOARCH=arm64 go build -o prepare-after-updater-macos-arm64
```
</div>


#### **Full Platform List**
```bash
go tool dist list
```

---

## **5. Build Verification**
### **5.1. Basic Execution Test**
```bash
./prepare-after-updater --help
```
Should display usage help.

### **5.2. Test Config Generation**
```bash
./prepare-after-updater --autoconfig test-config.json
```
- Verifies config template generation

### **5.3. Installation Test**
```bash
sudo ./prepare-after-updater --user $(whoami) --download file://$(pwd)/test-config.json
```
- Processes current user with local config

---

## **6. System Installation**
### **6.1. Binary Deployment**
```bash
sudo cp prepare-after-updater /usr/local/bin/
```
Now globally accessible:
```bash
prepare-after-updater --version
```

### **6.2. Systemd Service Creation**
For automatic startup, create `/etc/systemd/system/prepare-after-updater.service`:
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
Then:
```bash
sudo systemctl daemon-reload
sudo systemctl enable prepare-after-updater
sudo systemctl start prepare-after-updater
```

---

## **7. Package Creation (Optional)**
### **7.1. Debian/Ubuntu (DEB)**
1. Install `dpkg-deb`:
   ```bash
   sudo apt install dpkg-dev
   ```
2. Create package structure:
   ```bash
   mkdir -p deb-package/usr/local/bin
   mkdir -p deb-package/etc/prepare-after-updater
   cp prepare-after-updater deb-package/usr/local/bin/
   cp config.json deb-package/etc/prepare-after-updater/
   ```
3. Create `deb-package/DEBIAN/control`:
   ```plaintext
   Package: prepare-after-updater
   Version: 2.1.4
   Section: utils
   Priority: optional
   Architecture: amd64
   Maintainer: Your Name <your@email.com>
   Description: Post-update environment configuration tool.
   ```
4. Build package:
   ```bash
   dpkg-deb --build deb-package prepare-after-updater_2.1.4_amd64.deb
   ```

### **7.2. CentOS/RHEL (RPM)**
1. Install `rpm-build`:
   ```bash
   sudo yum install rpm-build
   ```
2. Create package structure:
   ```bash
   mkdir -p rpm-package/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
   ```
3. Create `rpm-package/SPECS/prepare-after-updater.spec`:
   ```plaintext
   Name: prepare-after-updater
   Version: 2.1.4
   Release: 1
   Summary: Post-update environment configuration tool.
   License: MIT
   URL: https://github.com/your-repo/prepare-after-updater
   Source0: prepare-after-updater
   BuildArch: x86_64

   %description
   Tool for automated user environment configuration after system updates.

   %install
   mkdir -p %{buildroot}/usr/local/bin
   mkdir -p %{buildroot}/etc/prepare-after-updater
   install -m 755 %{SOURCE0} %{buildroot}/usr/local/bin/
   install -m 644 config.json %{buildroot}/etc/prepare-after-updater/

   %files
   /usr/local/bin/prepare-after-updater
   /etc/prepare-after-updater/config.json
   ```
4. Build package:
   ```bash
   rpmbuild -bb rpm-package/SPECS/prepare-after-updater.spec
   ```

---

## **8. Build Validation**
### **8.1. Clean System Testing**
1. Deploy clean VM (Ubuntu/CentOS)
2. Install package:
   ```bash
   sudo dpkg -i prepare-after-updater_2.1.4_amd64.deb  # Debian
   sudo rpm -ivh prepare-after-updater-2.1.4-1.x86_64.rpm  # CentOS
   ```
3. Verify operation:
   ```bash
   sudo prepare-after-updater --version
   ```

### **8.2. Integration Tests**
Create `main_test.go`:
```go
package main

import (
	"testing"
)

func TestConfigLoading(t *testing.T) {
	cfg, err := loadAppConfig("testdata/config.json")
	if err != nil {
		t.Fatalf("Config load error: %v", err)
	}
	if cfg.HomeDir == "" {
		t.Error("HomeDir must not be empty")
	}
}
```
Run tests:
```bash
go test -v
```

---

## **9. Production Deployment**
1. Upload packages to repository (S3/Artifactory)
2. Configure automated installation (Ansible example):
   ```yaml
   - name: Install prepare-after-updater
     apt:
       deb: "https://your-repo/prepare-after-updater_2.1.4_amd64.deb"
   ```

---

## **10. Troubleshooting**
| Issue | Solution |
|-------|----------|
| `go: cannot find main module` | Run `go mod init` |
| No write permissions for `/usr/local/bin` | Use `sudo` |
| Error `undefined: http.Get` | Verify `GO111MODULE=on` |
| Program won't start after install | Check `ldd` (Linux) or `chmod +x` |

---

### **Summary**
The program builds in **3 stages**:
1. `go build` - binary compilation
2. `go test` - functionality verification
3. `dpkg-deb`/`rpmbuild` - optional packaging

Use `GOOS`/`GOARCH` for cross-platform support. Distribute final packages via DEB/RPM repositories.