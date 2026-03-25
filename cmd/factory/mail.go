package main

import (
	"github.com/spf13/cobra"
)

// getMailCmds returns mail/communication commands
func getMailCmds() []*cobra.Command {
	return []*cobra.Command{
		mailSendCmd,
		mailBroadcastCmd,
		mailListCmd,
	}
}

var mailSendCmd = &cobra.Command{
	Use:   "mail send <to> <message>",
	Short: "Send a message",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		to := args[0]
		// message := args[1]
		// TODO: Send message
		printSuccess("Message sent to %s", to)
		return nil
	},
}

var mailBroadcastCmd = &cobra.Command{
	Use:   "mail broadcast <message>",
	Short: "Broadcast to all stations",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		// message := args[0]
		// TODO: Broadcast message
		printSuccess("Message broadcasted to all stations")
		return nil
	},
}

var mailListCmd = &cobra.Command{
	Use:   "mail list",
	Short: "List messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		// TODO: List messages
		printInfo("No messages")
		return nil
	},
}
