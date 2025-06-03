package utils

import (
	"fmt"

	"github.com/spf13/cobra"
)

var action bool

var rootCmd = &cobra.Command{
	Use:   "broker",
	Short: "Broker - Go version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Init Broker")
	},
}

// Init the flags and calls the cobra to handle it
func Init() {
	rootCmd.Flags().BoolVarP(&action, "api", "", false, "Run the Broker API.")
	cobra.CheckErr(rootCmd.Execute())
}

// Gets the action selected by the user from the command line.
func Get_action_selected() bool {
	return action
}
