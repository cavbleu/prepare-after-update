# Description of the "Prepare After Updater" Program

The program is designed to automate the configuration of user environments after system updates or deployment of new machines. It handles configuration downloads, software installation, and settings customization for selected users.

**Documentation:**

[`Читать на Русском языке`](README_RU.md)  
[`Project Build`](docs/EN/README_compile.md)  
[`Program Mechanics`](docs/EN/README_Mechanisms.md)  

---

## **1. Core Functions and Workflow**

### **1.1. Initialization and Argument Processing**
- **Command-line flag parsing**:
  - `--version` - displays program version 
  - `--help` - shows usage instructions
  - `--home` - specifies path to user home directories (default: `/home`)
  - `--exclude` - list of user prefixes to exclude (e.g., `a_,adminsec`)
  - `--user` - processes a specific user (by name)
  - `--config` - path to configuration file (default: `/etc/prepare-after-updater/config.json`)
  - `--download` - name of downloadable configuration file (default: `web_cfg.json`)
  - `--log` - path to log file (default: `/var/log/prepare-after-updater.log`)
  - `--autoconfig` - generates a configuration template and exits

- **Privilege check**:
  - Requires `root` privileges (`os.Geteuid() != 0` → error).

---

### **1.2. Configuration Loading and Processing**
#### **1.2.1. Application Configuration (`AppConfig`)**
Structure:
```json
{
  "resource_url": "URL or file:// path to configuration",
  "home_dir": "/home",
  "log_path": "/var/log/prepare-after-updater.log",
  "exclude_prefixes": ["a_", "adminsec"]
}
```
- If no config is specified, defaults are used.
- Command-line flags override config parameters.

#### **1.2.2. Software Configuration (`ProgramConfig`)**
Structure:
```json
{
  "programs": [
    {
      "name": "Program name",
      "config_paths": [".config/app"],
      "check_command": "command to verify installation",
      "action": "install | execute",
      "packages": {"apt": "package1 package2", "yum": "package1 package2"},
      "command": "command to execute",
      "post_action": ["command1", "command2"]
    }
  ]
}
```
- Loaded from `web_cfg.json` (locally or via URL).
- Supports loading from:
  - HTTP (`http://example.com/config.json`)
  - Local files (`file:///path/to/config.json`)

---

### **1.3. User Selection for Processing**
1. If `--user` is specified, only their home directory is processed.
2. If no user is specified:
   - Scans `home_dir` (default: `/home`).
   - Excludes folders with prefixes from `exclude_prefixes`.
   - Displays interactive user selection menu.

---

### **1.4. Program Processing (Core Workflow)**
For each program in the config:
1. **Configuration check**:
   - Searches files in `config_paths` (e.g., `~/.config/app`).
   - If files exist → program is considered configured.

2. **Action determination**:
   - If `action` is unspecified:
     - If config exists → `action = "execute"`.
     - If no config → `action = "install"`.

3. **Action execution**:
   - **Installation (`install`)**:
     - Verifies program installation (`check_command`).
     - If not installed → installs via package manager (`apt`, `apt-get`, `yum`, `dnf`).
   - **Execution (`execute`)**:
     - Runs the `command`.
   - **Post-actions (`post_action`)**:
     - Executes additional commands after installation/configuration.

4. **Parallel processing**:
   - Programs are processed concurrently (goroutines + `sync.WaitGroup`).

---

### **1.5. Additional Features**
- **Package database update**:
  - Automatically detects package manager (`apt`, `apt-get`, `yum`, `dnf`).
  - Executes `apt update` / `yum check-update`.
- **Logging**:
  - Writes to file (`/var/log/prepare-after-updater.log`) and `stdout`.
  - Format: `[date] [time] message`.
- **Config template generation**:
  - `--autoconfig output.json` → creates a JSON template for manual editing.

---

## **2. Usage Examples**
### **2.1. Installing Software for User `user1`**
```bash
sudo prepare-after-updater --user user1 --download https://example.com/config.json
```
1. Downloads `config.json` from URL.
2. Applies settings only to `/home/user1`.

### **2.2. Interactive User Selection**
```bash
sudo prepare-after-updater --config /etc/custom-config.json
```
1. Loads config from `/etc/custom-config.json`.
2. Displays user list (excluding `a_*`, `adminsec*`).
3. Processes selected user.

### **2.3. Template Generation**
```bash
prepare-after-updater --autoconfig myconfig.json
```
Creates `myconfig.json` with sample program configuration.

---

## **3. Error Handling and Key Considerations**
- **Configuration loading errors**:
  - If URL is unreachable → program exits with error.
  - If local file is missing → uses default config.
- **Dependency checks**:
  - If no package manager is found → manual installation only.
- **Permissions**:
  - Requires `root` for package installation and `/home` access.
- **Temporary files**:
  - Downloaded `web_cfg.json` is deleted after processing.

---

## **4. Detailed Logic (Pseudocode)**
```plaintext
1. Parse arguments
2. If --version → display version and exit
3. If --help → display help and exit
4. If --autoconfig → generate template and exit
5. Initialize logger
6. Check privileges (root only)
7. Load configuration (flags > config > defaults)
8. Update package database (if package manager available)
9. User selection:
   - If --user → process only specified user
   - Else → interactive selection
10. For selected user:
    - Download program config (if URL provided)
    - For each program:
      - Determine action (install/execute)
      - Install or execute command
      - Run post-actions
11. Close log file and exit
```

---
### **Summary**
The program automates:
- Software installation via package managers.
- User environment configuration.
- Post-installation command execution.

Flexibility is ensured by:
- JSON configuration support.
- URL-based config loading.
- Parallel task execution.
- Advanced logging.
