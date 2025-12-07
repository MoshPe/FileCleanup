package cmd

import (
	"FileCleanup/pkg"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var AppConfig pkg.Config

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "FileCleanup",
	Short:   "A powerful tool for clean files based on their their relative size and modification date. ",
	Version: "1.0.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	pkg.InitConfigDir()
	pkg.InitConfigFile()
	initConfig()
	InitLogger()
	firstRunSetup()

	RootCmd.CompletionOptions.DisableDefaultCmd = true
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		viper.AddConfigPath(filepath.Join(home, ".fileCleanup"))
		viper.SetConfigName(".fileCleanup")
		viper.SetConfigType("json")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	configMap := viper.AllSettings()

	_, err := json.Marshal(configMap)
	if err != nil {
		fmt.Println("Error marshaling config map:", err)
		return
	}

	// Unmarshal the JSON into Config struct
	//err = json.Unmarshal(jsonData, &AppConfig)
	//if err != nil {
	//	fmt.Println("Error unmarshalling config:", err)
	//	return
	//}
}

func firstRunSetup() {
	home, _ := os.UserHomeDir()
	marker := filepath.Join(home, ".FileCleanup_setup_done")

	if _, err := os.Stat(marker); err == nil {
		return // already configured
	}

	shell := pkg.DetectShell()

	switch shell {
	case "bash":
		setupBash(home)
	case "zsh":
		setupZsh(home)
	case "fish":
		setupFish(home)
	case "pwsh", "powershell":
		setupPowershell(home)
	case "cmd":
		fmt.Println("cmd.exe doesn't support tab-completion; use PowerShell instead.")
	default:
		fmt.Println("Unknown shell, skipping completion setup")
	}

	// Create marker file
	os.WriteFile(marker, []byte("ok"), 0o644)
}

func setupBash(home string) {
	rc := filepath.Join(home, ".bashrc")
	line := "source <(FileCleanup completion bash)"

	appendIfMissing(rc, line)
}

func setupZsh(home string) {
	// Create completions folder
	comp := filepath.Join(home, ".zsh/completions")
	os.MkdirAll(comp, 0o755)

	// Install completion script
	script := filepath.Join(comp, "_FileCleanup")
	f, _ := os.Create(script)
	RootCmd.GenZshCompletion(f)
	f.Close()

	// Ensure fpath is updated
	rc := filepath.Join(home, ".zshrc")
	line := `fpath=(~/.zsh/completions $fpath)`
	appendIfMissing(rc, line)

	appendIfMissing(rc, "autoload -Uz compinit")
	appendIfMissing(rc, "compinit")
}

func setupFish(home string) {
	folder := filepath.Join(home, ".config/fish/completions")
	os.MkdirAll(folder, 0o755)

	file := filepath.Join(folder, "FileCleanup.fish")
	f, _ := os.Create(file)
	RootCmd.GenFishCompletion(f, true)
	f.Close()
}

func setupPowershell(home string) {
	profile := filepath.Join(home, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	os.MkdirAll(filepath.Dir(profile), 0o755)

	line := "FileCleanup completion powershell | Out-String | Invoke-Expression"
	appendIfMissing(profile, line)
}

func appendIfMissing(path, line string) {
	data, _ := os.ReadFile(path)

	if strings.Contains(string(data), line) {
		return
	}

	f, _ := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	defer f.Close()

	f.WriteString("\n" + line + "\n")
}
