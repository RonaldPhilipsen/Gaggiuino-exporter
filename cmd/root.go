package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/RonaldPhilipsen/gaggiuino-exporter/build"
	gaggiuino "github.com/RonaldPhilipsen/gaggiuino-exporter/gagguino"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rootCmd = &cobra.Command{
	Use:   "gaggiuino_exporter",
	Short: "Prometheus exporter for Gaggiuino espresso machine",
	Long: `Prometheus exporter for Gaggiuino espresso machine.`,
	Run: func(cmd *cobra.Command, args []string) {
		printHeader()
		
		baseURL := gaggiuino.GetBaseUrl()

		if viper.GetString("mode") == "state" {
			lastShotID, err := gaggiuino.GetLastShot(baseURL)
			if err != nil {
				log.Fatalf("Error getting last shot ID: %v", err)
			}
			state, err := gaggiuino.GetState(baseURL)
			if err != nil {
				log.Fatalf("Error getting state: %v", err)
			}
			stateJSON, err := json.Marshal(state)
			if err != nil {
				log.Fatalf("Error marshaling state: %v", err)
			}
			log.Printf("current state: %s, last shot ID: %d", string(stateJSON), lastShotID)
			os.Exit(0)
		}
		// TODO: Log all shots to Postgres
		e := gaggiuino.NewExporter(
			baseURL, 
			viper.GetStringMapString("basic-auth-users"),
		)
		e.RunServer(viper.GetString("bind-address"))
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	flags := rootCmd.PersistentFlags()
	flags.StringP("mode", "m", "http", "Expose method - either 'state' or 'http'")
	flags.StringP("bind-address", "b", "0.0.0.0:9995", "Address to bind to")
	flags.StringToString("basic-auth-users", map[string]string{}, "Basic Auth users and their passwords as bcypt hashes")
	err := viper.BindPFlags(flags)
	if err != nil {
		log.Fatalf("Error binding flags: %v", err)
	}
}


// initConfig reads in  ENV variables if set.
func initConfig() {
	// TODO: figure out how to map env variables.
	viper.AutomaticEnv() // read in environment variables that match
}

func printHeader() {
	log.Printf("gaggiuino-exporter %s	build date: %s	sha1: %s	Go: %s	GOOS: %s	GOARCH: %s",
		build.BuildVersion,
		build.BuildDate,
		build.BuildCommitSha,
		runtime.Version(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

