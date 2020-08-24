package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

func SetDefaults() {
	viper.SetDefault("buildConcurrently", false)
	viper.SetDefault("prepareContainers", false)
	viper.SetDefault("cleanupInterval", 30)
	viper.SetDefault("defaultLanguage.memory", "256mb")
	viper.SetDefault("defaultLanguage.cpus", 0.25)
	viper.SetDefault("defaultLanguage.timeout", 20)
	viper.SetDefault("defaultLanguage.concurrent", 5)
	viper.SetDefault("defaultLanguage.retries", 10)
	viper.SetDefault("defaultLanguage.outputLimit", "4kb")
	viper.SetDefault("languages_path", "./languages")
}

func UseConfigFile(path string) {
	viper.SetConfigFile(path)
}

func TryFindConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.myriag")
	viper.AddConfigPath("/etc/myriag")
	viper.WatchConfig()
}

func ReadInConfig() error {
	return viper.ReadInConfig()
}

func ConfigFileUsed() string {
	return viper.ConfigFileUsed()
}

func BuildConcurrently() bool {
	return viper.GetBool("buildConcurrently")
}

func PrepareContainers() bool {
	return viper.GetBool("prepareContainers")
}

func CleanupInterval() time.Duration {
	return time.Minute * time.Duration(viper.GetInt("cleanupInterval"))
}

func Port() string {
	return viper.GetString("port")
}

func Host() string {
	return viper.GetString("host")
}

func Languages() []string {
	res := make([]string, 0)
	languages := viper.GetStringMap("languages")
	for language := range languages {
		res = append(res, language)
	}

	return res
}

func PathToLanguages() string {
	return viper.GetString("languages_path")
}

func MaxConcurrentEvlasFor(lang string) int {
	key := fmt.Sprintf("languages.%s.concurrent", lang)
	if viper.IsSet(key) {
		return viper.GetInt(key)
	} else {
		return viper.GetInt("defaultLanguage.concurrent")
	}
}

func MemoryFor(lang string) int64 {
	key := fmt.Sprintf("languages.%s.memory", lang)
	if viper.IsSet(key) {
		return int64(viper.GetSizeInBytes(key))
	} else {
		return int64(viper.GetSizeInBytes("defaultLanguage.memory"))
	}
}

func NanoCPUFor(lang string) int64 {
	key := fmt.Sprintf("languages.%s.cpus", lang)
	if viper.IsSet(key) {
		return int64(10e9 * viper.GetFloat64(key))
	} else {
		return int64(10e9 * viper.GetFloat64("defaultLanguage.cpus"))
	}
}

func RetryCountFor(lang string) int {
	key := fmt.Sprintf("languages.%s.retries", lang)
	if viper.IsSet(key) {
		return viper.GetInt(key)
	} else {
		return viper.GetInt("defaultLanguage.retries")
	}
}

func MaxOutputFor(lang string) (size uint) {
	key := fmt.Sprintf("languages.%s.outputLimit", lang)
	if viper.IsSet(key) {
		size = viper.GetSizeInBytes(key)
	} else {
		size = viper.GetSizeInBytes("defaultLanguage.outputLimit")
	}
	return size
}

func TimeoutFor(lang string) time.Duration {
	key := fmt.Sprintf("languages.%s.timeout", lang)
	if viper.IsSet(key) {
		return time.Second * viper.GetDuration(key)
	} else {
		return time.Second * viper.GetDuration("defaultLanguage.timeout")
	}
}

func IsLangSupported(lang string) bool {
	exists := false
	for _, supportedLanguage := range Languages() {
		if supportedLanguage == lang {
			exists = true
			break
		}
	}
	return exists
}
