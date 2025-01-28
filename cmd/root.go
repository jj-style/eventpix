package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/adrg/xdg"
	"github.com/jj-style/eventpix/internal/config"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	cfg     = new(config.Config)

	Version string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "eventpix",
	Short:   "Backend of eventpix",
	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
}

func initConfig() {
	// Don't forget to read config either from cfgFile or from home directory!
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath("/etc/eventpix/") // path to look for the config file in
		viper.AddConfigPath(filepath.Join(xdg.ConfigHome, "eventpix"))
		viper.AddConfigPath(".") // optionally look for config in the working directory
	}
	viper.SetDefault("nats.url", nats.DefaultURL)

	viper.SetEnvPrefix("eventpix")
	viper.EnvKeyReplacer(strings.NewReplacer(`.`, `_`))
	viper.AutomaticEnv()
	// https://github.com/spf13/viper/issues/761#issuecomment-1578931559
	for _, e := range os.Environ() {
		split := strings.Split(e, "=")
		k := split[0]

		if strings.HasPrefix(k, strings.ToUpper(viper.GetEnvPrefix())) {
			viper.BindEnv(strings.Join(strings.Split(k, "_")[1:], "."), k)
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Can't read config:", err)
		os.Exit(1)
	}
	if err := viper.Unmarshal(cfg); err != nil {
		fmt.Println("Can't parse config:", err)
		os.Exit(1)
	}
}
