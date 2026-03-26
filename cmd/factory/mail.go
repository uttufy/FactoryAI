package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uttufy/FactoryAI/internal/beads"
)

// getMailCmd returns the mail parent command with all subcommands
func getMailCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mail",
		Short: "Mail/communication commands",
	}
	cmd.AddCommand(mailSendCmd, mailBroadcastCmd, mailListCmd)
	return cmd
}

var mailSendCmd = &cobra.Command{
	Use:   "send <to> <subject> <body>",
	Short: "Send a message",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		if mailSystem == nil {
			return fmt.Errorf("mail system not initialized")
		}

		ctx := context.Background()
		to := args[0]
		subject := args[1]
		body := args[2]

		msg := &beads.Message{
			From:      "director",
			To:        to,
			Subject:   subject,
			Body:      body,
			Type:      beads.MsgNotify,
			Priority:  5,
		}

		if err := mailSystem.Send(ctx, msg); err != nil {
			return fmt.Errorf("sending message: %w", err)
		}

		printSuccess("Message sent to %s", to)
		return nil
	},
}

var mailBroadcastCmd = &cobra.Command{
	Use:   "broadcast <subject> <body>",
	Short: "Broadcast to all stations",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		if mailSystem == nil {
			return fmt.Errorf("mail system not initialized")
		}

		ctx := context.Background()
		subject := args[0]
		body := args[1]

		if err := mailSystem.Broadcast(ctx, "director", subject, body); err != nil {
			return fmt.Errorf("broadcasting message: %w", err)
		}

		printSuccess("Message broadcast to all stations")
		return nil
	},
}

var mailListCmd = &cobra.Command{
	Use:   "list [station]",
	Short: "List messages",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireBoot(); err != nil {
			return err
		}

		if mailSystem == nil {
			return fmt.Errorf("mail system not initialized")
		}

		ctx := context.Background()
		stationID := "all"
		if len(args) > 0 {
			stationID = args[0]
		}

		if stationID == "all" {
			// List mail for all stations
			stations := stationManager.List(ctx)
			for _, s := range stations {
				messages, err := mailSystem.Receive(ctx, s.ID)
				if err != nil {
					continue
				}
				if len(messages) > 0 {
					fmt.Printf("\nStation %s:\n", s.ID)
					for _, msg := range messages {
						read := "unread"
						if msg.Read {
							read = "read"
						}
						fmt.Printf("  [%s] %s: %s (%s)\n", read, msg.From, msg.Subject, msg.Timestamp.Format("15:04:05"))
					}
				}
			}
		} else {
			messages, err := mailSystem.Receive(ctx, stationID)
			if err != nil {
				return fmt.Errorf("reading mail: %w", err)
			}

			if len(messages) == 0 {
				printInfo("No messages for station %s", stationID)
				return nil
			}

			fmt.Printf("Messages for station %s:\n", stationID)
			for _, msg := range messages {
				read := "unread"
				if msg.Read {
					read = "read"
				}
				fmt.Printf("  [%s] %s: %s\n", read, msg.From, msg.Subject)
				fmt.Printf("    %s\n", msg.Body)
				fmt.Printf("    (%s)\n", msg.Timestamp.Format("2006-01-02 15:04:05"))
			}
		}
		return nil
	},
}
