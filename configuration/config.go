// Package configuration manages configuration loading from environment variable. If not in environment variable then it loads it from included yaml file.
package configuration

import (
	"flag"
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config is the configuration type for Micoservice.
type Config struct {
	Server struct {
		Port                int    `yaml:"port" env:"DEMOSERVER_CONNECTIONMANAGER_SERVER_PORT"`
		PrefixMain          string `yaml:"prefix_main" env:"DEMOSERVER_CONNECTIONMANAGER_PREFIX_MAIN"`
		PrefixWorker        string `yaml:"prefix_worker" env:"DEMOSERVER_CONNECTIONMANAGER_PREFIX_WORKER"`
		HTTPReadTimeout     int    `yaml:"http_read_timeout" env:"DEMOSERVER_CONNECTIONMANAGER_HTTP_READ_TIMEOUT"`
		HTTPWriteTimeout    int    `yaml:"http_write_timeout" env:"DEMOSERVER_CONNECTIONMANAGER_HTTP_WRITE_TIMEOUT"`
		HTTPIdleTimeout     int    `yaml:"http_idle_timeout" env:"DEMOSERVER_CONNECTIONMANAGER_HTTP_IDLE_TIMEOUT"`
		HTTPShutdownTimeout int    `yaml:"http_shutdown_timeout" env:"DEMOSERVER_CONNECTIONMANAGER_HTTP_SHUTDOWN_TIMEOUT"`
		WokerSleepTime      int    `yaml:"worker_sleep_time" env:"DEMOSERVER_CONNECTIONMANAGER_WORKER_SLEEP_TIME"`
		ListLimit           int    `yaml:"list_limit" env:"DEMOSERVER_CONNECTIONMANAGER_LIST_LIMIT"`
	} `yaml:"server"`

	Configuration struct {
		RefreshCycle int    `yaml:"refresh_cycle" env:"DEMOSERVER_CONNECTIONMANAGER_CONFIGURATION_REFRESH_CYCLE"`
		LogFolder    string `yaml:"log_folder" env:"DEMOSERVER_CONNECTIONMANAGER_CONFIGURATION_LOG_FOLDER"`
		LogFile      string `yaml:"log_file" env:"DEMOSERVER_CONNECTIONMANAGER_CONFIGURATION_LOG_FILE"`
	} `yaml:"configuration"`

	Postgres struct {
		Host                 string `yaml:"host" env:"DEMOSERVER_CONNECTIONMANAGER_POSTGRES_HOST"`
		Port                 int    `yaml:"port" env:"DEMOSERVER_CONNECTIONMANAGER_POSTGRES_PORT"`
		ROUsername           string `yaml:"rousername" env:"DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_USERNAME"`
		RWUsername           string `yaml:"rwusername" env:"DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_USERNAME"`
		ROPassword           string `yaml:"ropassword" env:"DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_PASSWORD"`
		RWPassword           string `yaml:"rwpassword" env:"DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_PASSWORD"`
		ROConnectionPoolSize int    `yaml:"roconnectionpoolsize" env:"DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RO_CONNECTIONPOOLSIZE"`
		RWConnectionPoolSize int    `yaml:"rwconnectionpoolsize" env:"DEMOSERVER_CONNECTIONMANAGER_POSTGRES_RW_CONNECTIONPOOLSIZE"`
		SSLMode              bool   `yaml:"sslmode" env:"DEMOSERVER_CONNECTIONMANAGER_POSTGRES_SSLMODE"`
	} `yaml:"postgres"`

	Vault struct {
		Host          string `yaml:"host" env:"DEMOSERVER_CONNECTIONMANAGER_VAULT_HOST"`
		Port          int    `yaml:"port" env:"DEMOSERVER_CONNECTIONMANAGER_VAULT_PORT"`
		RoleID        string `yaml:"roleid" env:"DEMOSERVER_CONNECTIONMANAGER_VAULT_ROLE_ID"`
		SecretID      string `yaml:"secretid" env:"DEMOSERVER_CONNECTIONMANAGER_VAULT_SECRET_ID"`
		HTTPS         bool   `yaml:"https" env:"DEMOSERVER_CONNECTIONMANAGER_VAULT_HTTPS"`
		TLSSkipVerify bool   `yaml:"tlsskipverify" env:"DEMOSERVER_CONNECTIONMANAGER_VAULT_TLSSKIPVERIFY"`
		PathPrefix    string `yaml:"pathprefix" env:"DEMOSERVER_CONNECTIONMANAGER_VAULT_PATH_PREFIX"`
	} `yaml:"vault"`

	DataLayer struct {
		NamePrefix string `yaml:"name_prefix" env:"DEMOSERVER_CONNECTIONMANAGER_DATALAYER_NAME_PREFIX"`
		MaxResults int    `yaml:"max_results" env:"DEMOSERVER_CONNECTIONMANAGER_DATALAYER_MAX_RESULTS"`
	} `yaml:"datalayer"`

	AWS struct {
		DefaultLeaseTTL int `yaml:"default_lease_ttl" env:"DEMOSERVER_CONNECTIONMANAGER_AWS_DEFAULTLEASETTL"`
		MaxLeaseTTL     int `yaml:"max_lease_ttl" env:"DEMOSERVER_CONNECTIONMANAGER_AWS_MAXLEASETTL"`
	} `yaml:"aws"`
}

// Args is the struct for pass .
type Args struct {
	ConfigPath string
}

// ProcessArgs figures out Config yaml files path, loads it and returns its path to caller.
func ProcessArgs(cfg interface{}) string {
	var ConfigPath string

	f := flag.NewFlagSet("DEMOSERVER_CONNECTIONMANAGER", 1)
	f.StringVar(&ConfigPath, "c", "demoserver_connectionmanager_env_config.yml", "./")

	fu := f.Usage
	f.Usage = func() {
		fu()
		envHelp, _ := cleanenv.GetDescription(cfg, nil)
		fmt.Fprintln(f.Output())
		fmt.Fprintln(f.Output(), envHelp)
	}

	err := f.Parse(os.Args[1:])

	if err != nil {
		fmt.Println(err.Error())
	}

	return ConfigPath
}
