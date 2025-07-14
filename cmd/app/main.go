package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	defaultVersion    = "2.1.4"
	defaultConfigDir  = "/etc/prepare-after-updater"
	defaultConfigFile = "config.json"
	defaultDownloaded = "web_cfg.json"
	defaultHomeDir    = "/home"
	defaultLogPath    = "/var/log/prepare-after-updater.log"
	defaultExclude    = "a_,adminsec"
)

type AppConfig struct {
	ResourceURL string   `json:"resource_url"`
	HomeDir     string   `json:"home_dir"`
	LogPath     string   `json:"log_path"`
	Exclude     []string `json:"exclude_prefixes"`
}

type ProgramConfig struct {
	Programs []Program `json:"programs"`
}

type Program struct {
	Name         string            `json:"name"`
	ConfigPaths  []string          `json:"config_paths"`
	CheckCommand string            `json:"check_command"`
	Action       string            `json:"action"`
	Packages     map[string]string `json:"packages"`
	Command      string            `json:"command"`
	PostAction   []string          `json:"post_action"`
}

var (
	versionFlag      = flag.Bool("version", false, "Показать версию программы")
	helpFlag         = flag.Bool("help", false, "Показать справку")
	homeDirFlag      = flag.String("home", "", "Путь к домашним директориям пользователей")
	excludeFlag      = flag.String("exclude", "", "Префиксы исключаемых папок (через запятую)")
	userFlag         = flag.String("user", "", "Обработать конкретного пользователя (по имени)")
	configPathFlag   = flag.String("config", "", "Путь к конфигурационному файлу")
	downloadPathFlag = flag.String("download", "", "Имя скачиваемого файла конфигурации")
	logPathFlag      = flag.String("log", "", "Путь к файлу логов")
	autoconfigFlag   = flag.String("autoconfig", "", "Сгенерировать шаблон конфигурации и выйти")

	logger *log.Logger
)

func printHelp() {
	fmt.Println("Использование:")
	fmt.Println("  prepare-after-updater [опции]")
	fmt.Println("\nОпции:")
	flag.PrintDefaults()
	fmt.Println("\t==== Примеры: ====")
	fmt.Println("  1. Использование строковых флагов")
	fmt.Println("  program --home /home --exclude a_,test --logout /var/log/update.log")
	fmt.Println("  program --download https://my.site/config.json")
	fmt.Println("  2. Использование локального конфигурационного файла config.json")
	fmt.Println("  3. Использование параметров по умолчанию")
}

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Prepare After Updater v%s\n", defaultVersion)
		os.Exit(0)
	}

	if *autoconfigFlag != "" {
		generateConfigTemplate(*autoconfigFlag)
		os.Exit(0)
	}

	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	initLogging()
	defer logger.Println("=== Завершение работы программы ===")

	logger.Println("=== Запуск программы ===")
	logger.Printf("Версия программы: %s", defaultVersion)

	if os.Geteuid() != 0 {
		logger.Fatal("Программа должна запускаться с правами root!")
	}

	// Если конфиг не указан и не найден по умолчанию, и флаги не заданы - используем параметры по умолчанию
	if *configPathFlag == "" {
		if _, err := os.Stat(filepath.Join(defaultConfigDir, defaultConfigFile)); os.IsNotExist(err) && flagsNotSet() {
			logger.Println("Конфигурационный файл не найден и флаги не заданы - использование параметров по умолчанию")
			cfg := &AppConfig{
				HomeDir: defaultHomeDir,
				LogPath: defaultLogPath,
				Exclude: strings.Split(defaultExclude, ","),
			}
			processUsers(cfg)
			return
		}
	}

	cfg, err := loadAppConfig()
	if err != nil {
		logger.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	if err := updatePackageDatabase(); err != nil {
		logger.Printf("Внимание: %v\n", err)
	}

	processUsers(cfg)
}

// Проверяет, заданы ли какие-либо флаги (кроме version, help и autoconfig)
func flagsNotSet() bool {
	return *homeDirFlag == "" && *excludeFlag == "" && *userFlag == "" &&
		*downloadPathFlag == "" && *logPathFlag == ""
}

func generateConfigTemplate(path string) {
	template := ProgramConfig{
		Programs: []Program{
			{
				Name:        "Имя программы",
				ConfigPaths: []string{".config/app"},
				Action:      "install",
				Packages: map[string]string{
					"apt": "пакет1 пакет2",
					"yum": "пакет1 пакет2",
				},
				Command:    "команда для выполнения",
				PostAction: []string{"команда1", "команда2"},
			},
			{
				Name: "Имя программы",
				ConfigPaths: []string{
					".config1",
					".config2"},
				Action:     "execute",
				Command:    "команда для выполнения",
				PostAction: []string{"команда1", "команда2"},
			},
		},
	}

	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		fmt.Printf("Ошибка генерации шаблона: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Printf("Ошибка записи файла: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Шаблон конфигурации создан: %s\n", path)
}

func updatePackageDatabase() error {
	logger.Println("Обновление пакетной базы...")

	var updateCmd *exec.Cmd
	switch {
	case commandExists("apt-get"):
		updateCmd = exec.Command("apt-get", "update")
	case commandExists("apt"):
		updateCmd = exec.Command("apt", "update")
	case commandExists("yum"):
		updateCmd = exec.Command("yum", "check-update")
	case commandExists("dnf"):
		updateCmd = exec.Command("dnf", "check-update")
	default:
		return fmt.Errorf("не найдено поддерживаемого пакетного менеджера")
	}

	updateCmd.Stdout = logger.Writer()
	updateCmd.Stderr = logger.Writer()

	if err := updateCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 100 {
			logger.Println("Доступны обновления пакетов")
			return nil
		}
		return fmt.Errorf("Обновление базы данных пакетов не удалось: %v", err)
	}

	logger.Println("База данных пакетов успешно обновлена")
	return nil
}

func initLogging() {
	logPath := defaultLogPath
	if *logPathFlag != "" {
		logPath = *logPathFlag
	}

	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Ошибка создания директории логов: %v\n", err)
		os.Exit(1)
	}

	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Ошибка открытия файла логов: %v\n", err)
		os.Exit(1)
	}

	logger = log.New(io.MultiWriter(os.Stdout, logFile), "", log.LstdFlags)
}

func loadAppConfig() (*AppConfig, error) {
	configPath := filepath.Join(defaultConfigDir, defaultConfigFile)
	if *configPathFlag != "" {

		configPath = *configPathFlag
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла конфигурации: %v", err)
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка разбора конфигурации: %v", err)
	}

	// Применяем флаги
	if *homeDirFlag != "" {
		cfg.HomeDir = *homeDirFlag
	} else if cfg.HomeDir == "" {
		cfg.HomeDir = defaultHomeDir
	}

	if *logPathFlag != "" {
		cfg.LogPath = *logPathFlag
	} else if cfg.LogPath == "" {
		cfg.LogPath = defaultLogPath
	}

	if *excludeFlag != "" {
		cfg.Exclude = strings.Split(*excludeFlag, ",")
	} else if len(cfg.Exclude) == 0 {
		cfg.Exclude = strings.Split(defaultExclude, ",")
	}

	return &cfg, nil
}

func selectUserHome(homeDir string, exclude []string) (string, error) {
	entries, err := os.ReadDir(homeDir)
	if err != nil {
		return "", fmt.Errorf("ошибка чтения директории %s: %v", homeDir, err)
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			if !hasExcludedPrefix(name, exclude) {
				dirs = append(dirs, name)
			} else {
				logger.Printf("Пропуск исключенной директории: %s", name)
			}
		}
	}

	if len(dirs) == 0 {
		return "", fmt.Errorf("в директории %s нет подходящих папок", homeDir)
	}

	fmt.Println("Выберите домашнюю папку пользователя:")
	for i, dir := range dirs {
		fmt.Printf("%d. %s\n", i+1, dir)
	}

	var choice int
	fmt.Print("Введите номер: ")
	_, err = fmt.Scan(&choice)
	if err != nil || choice < 1 || choice > len(dirs) {
		return "", fmt.Errorf("неверный выбор")
	}

	return filepath.Join(homeDir, dirs[choice-1]), nil
}

func processUsers(cfg *AppConfig) {
	// Если указан конкретный пользователь через флаг --user
	if *userFlag != "" {
		userPath := filepath.Join(cfg.HomeDir, *userFlag)
		if _, err := os.Stat(userPath); err == nil {
			logger.Printf("Обработка указанного пользователя: %s", *userFlag)
			processUserConfig(userPath, cfg.ResourceURL)
			return
		}
		logger.Printf("Пользователь %s не найден, перехожу к выбору", *userFlag)
	}

	// Интерактивный выбор пользователя
	selectedHome, err := selectUserHome(cfg.HomeDir, cfg.Exclude)
	if err != nil {
		logger.Fatalf("Ошибка выбора пользователя: %v", err)
	}

	logger.Printf("Выбрана домашняя папка: %s", selectedHome)
	processUserConfig(selectedHome, cfg.ResourceURL)
}

func getFilteredUsers(homeDir string, exclude []string) ([]string, error) {
	var users []string

	entries, err := os.ReadDir(homeDir)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения директории %s: %v", homeDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			if !hasExcludedPrefix(name, exclude) {
				users = append(users, filepath.Join(homeDir, name))
			} else {
				logger.Printf("Пропуск исключенной директории: %s", name)
			}
		}
	}

	return users, nil
}

func hasExcludedPrefix(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func processUserConfig(userHome, resourceURL string) {
	downloadedFile := defaultDownloaded
	if *downloadPathFlag != "" {
		downloadedFile = *downloadPathFlag
	}

	configPath := filepath.Join(userHome, downloadedFile)
	if resourceURL != "" {
		if err := downloadConfig(resourceURL, configPath); err != nil {
			logger.Printf("Ошибка загрузки конфигурации для %s: %v", userHome, err)
			return
		}
		defer os.Remove(configPath)
	} else {
		logger.Println("URL ресурса не указан, пропуск загрузки конфигурации")
		return
	}

	cfg, err := loadProgramConfig(configPath)
	if err != nil {
		logger.Printf("Ошибка загрузки конфигурации программ: %v", err)
		return
	}

	for _, program := range cfg.Programs {
		processProgram(userHome, program)
	}
}

func downloadConfig(srcURL, dstPath string) error {
	logger.Printf("Загрузка конфигурации из %s", srcURL)

	if strings.HasPrefix(srcURL, "file://") {
		u, err := url.Parse(srcURL)
		if err != nil {
			return fmt.Errorf("неверный URL файла: %v", err)
		}

		srcPath := u.Path
		if u.Host != "" {
			srcPath = u.Host + u.Path
		}

		logger.Printf("Копирование локального файла из %s в %s", srcPath, dstPath)
		return copyLocalFile(srcPath, dstPath)
	}

	resp, err := http.Get(srcURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка загрузки: %s", resp.Status)
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func copyLocalFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("ошибка открытия исходного файла: %v", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("ошибка создания целевого файла: %v", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("ошибка копирования файла: %v", err)
	}

	return nil
}

func loadProgramConfig(path string) (*ProgramConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла конфигурации: %v", err)
	}

	var cfg ProgramConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка разбора конфигурации: %v", err)
	}

	return &cfg, nil
}

func processProgram(userHome string, program Program) {
	logger.Printf("Проверка программы: %s", program.Name)

	configExists := checkConfigExists(userHome, program.ConfigPaths)

	if program.Action == "" {
		if configExists {
			program.Action = "execute"
		} else {
			program.Action = "install"
		}
	}

	switch program.Action {
	case "install":
		if isProgramInstalled(program.CheckCommand) {
			logger.Printf("%s уже установлен", program.Name)
			return
		}

		if err := installProgram(program); err != nil {
			logger.Printf("Ошибка установки %s: %v", program.Name, err)
			return
		}

	case "execute":
		if !configExists {
			logger.Printf("Конфигурация для %s не найдена, пропуск выполнения", program.Name)
			return
		}

		if program.Command == "" {
			logger.Printf("Не указана команда для выполнения %s", program.Name)
			return
		}

		if err := executeCommand(program.Command); err != nil {
			logger.Printf("Ошибка выполнения команды для %s: %v", program.Name, err)
			return
		}

	default:
		logger.Printf("Неизвестное действие для %s: %s", program.Name, program.Action)
		return
	}

	runPostAction(program.PostAction)
}

func checkConfigExists(homeDir string, paths []string) bool {
	for _, path := range paths {
		fullPath := filepath.Join(homeDir, path)
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			return true
		}
	}
	return false
}

func isProgramInstalled(checkCommand string) bool {
	cmdParts := strings.Fields(checkCommand)
	if len(cmdParts) == 0 {
		return false
	}

	_, err := exec.LookPath(cmdParts[0])
	if err != nil {
		return false
	}

	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
	return cmd.Run() == nil
}

func installProgram(program Program) error {
	pm := detectPackageManager()
	if pm == "" {
		return fmt.Errorf("не найден поддерживаемый менеджер пакетов")
	}

	pkgList, ok := program.Packages[pm]
	if !ok {
		return fmt.Errorf("пакеты для %s не определены", pm)
	}

	logger.Printf("Установка с помощью %s: %s", pm, pkgList)

	cmd := exec.Command(pm, "install", "-y")
	cmd.Args = append(cmd.Args, strings.Fields(pkgList)...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Printf("Вывод команды установки:\n%s", string(output))
		return fmt.Errorf("ошибка установки: %v", err)
	}

	logger.Printf("Установка завершена успешно. Вывод:\n%s", string(output))
	return nil
}

func executeCommand(cmd string) error {
	logger.Printf("Выполнение команды: %s", cmd)
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return fmt.Errorf("пустая команда")
	}

	output, err := exec.Command(parts[0], parts[1:]...).CombinedOutput()
	if err != nil {
		logger.Printf("Вывод команды:\n%s", string(output))
		return fmt.Errorf("ошибка выполнения: %v", err)
	}

	logger.Printf("Команда выполнена успешно. Вывод:\n%s", string(output))
	return nil
}

func runPostAction(commands []string) {
	for _, cmd := range commands {
		logger.Printf("Выполнение пост-действия: %s", cmd)
		if err := executeCommand(cmd); err != nil {
			logger.Printf("Ошибка выполнения пост-действия: %v", err)
		}
	}
}

func detectPackageManager() string {
	switch {
	case commandExists("apt"):
		return "apt"
	case commandExists("apt-get"):
		return "apt-get"
	case commandExists("dnf"):
		return "dnf"
	case commandExists("yum"):
		return "yum"
	default:
		return ""
	}
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
