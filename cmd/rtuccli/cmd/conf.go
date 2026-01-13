package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/icegreg/chat-smpl/cmd/rtuccli/client"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var confCmd = &cobra.Command{
	Use:   "conf",
	Short: "Manage conferences",
	Long:  "Manage and monitor voice conferences",
}

var confListCmd = &cobra.Command{
	Use:   "list",
	Short: "List conferences",
	Long:  "List all conferences with optional status filter",
	Run: func(cmd *cobra.Command, args []string) {
		apiClient := client.NewAPIClient(serverURL)

		status, _ := cmd.Flags().GetString("status")

		conferences, err := apiClient.ListConferences(status)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(conferences) == 0 {
			fmt.Println("No conferences found")
			return
		}

		// Print table
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Conference ID", "Name", "Type", "Status", "Participants", "Duration", "Started"})
		table.SetBorder(true)
		table.SetRowLine(true)

		for _, conf := range conferences {
			duration := "-"
			if conf.Duration != nil {
				duration = formatDuration(*conf.Duration)
			}

			started := "-"
			if conf.StartedAt != nil {
				started = conf.StartedAt.Format("2006-01-02 15:04")
			}

			table.Append([]string{
				truncateString(conf.ID, 8),
				conf.Name,
				conf.EventType,
				conf.Status,
				fmt.Sprintf("%d", conf.Participants),
				duration,
				started,
			})
		}

		fmt.Printf("CONFERENCES (%d):\n\n", len(conferences))
		table.Render()
	},
}

var confGetCmd = &cobra.Command{
	Use:   "get [conference-id]",
	Short: "Get conference details",
	Long:  "Get detailed information about a specific conference",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiClient := client.NewAPIClient(serverURL)

		conf, err := apiClient.GetConference(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("CONFERENCE: %s\n", conf.ID)
		fmt.Printf("Name:         %s\n", conf.Name)
		fmt.Printf("Type:         %s\n", conf.EventType)
		fmt.Printf("Status:       %s\n", conf.Status)
		fmt.Printf("Participants: %d\n", conf.Participants)

		if conf.StartedAt != nil {
			fmt.Printf("Started:      %s\n", conf.StartedAt.Format("2006-01-02 15:04:05"))
		}

		if conf.Duration != nil {
			fmt.Printf("Duration:     %s\n", formatDuration(*conf.Duration))
		}

		if conf.ChatID != nil {
			fmt.Printf("Chat ID:      %s\n", *conf.ChatID)
		}

		fmt.Println()
		fmt.Printf("Use 'rtuccli conf clients %s' to see participants\n", conf.ID)
	},
}

var confClientsCmd = &cobra.Command{
	Use:   "clients [conference-id]",
	Short: "List conference participants",
	Long:  "List all participants (clients) of a conference",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiClient := client.NewAPIClient(serverURL)

		participants, err := apiClient.ListParticipants(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(participants) == 0 {
			fmt.Println("No participants found")
			return
		}

		// Print table
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Username", "Extension", "Status", "Joined", "Duration"})
		table.SetBorder(true)
		table.SetRowLine(true)

		for _, p := range participants {
			joined := "-"
			if p.JoinedAt != nil {
				joined = p.JoinedAt.Format("15:04:05")
			}

			duration := "-"
			if p.Duration != nil {
				duration = formatDuration(*p.Duration)
			}

			table.Append([]string{
				p.Username,
				p.Extension,
				p.Status,
				joined,
				duration,
			})
		}

		fmt.Printf("CONFERENCE PARTICIPANTS (%d):\n\n", len(participants))
		table.Render()
	},
}

func init() {
	rootCmd.AddCommand(confCmd)
	confCmd.AddCommand(confListCmd)
	confCmd.AddCommand(confGetCmd)
	confCmd.AddCommand(confClientsCmd)

	confListCmd.Flags().String("status", "", "Filter by status (active, scheduled, ended)")
}

func formatDuration(seconds int64) string {
	d := time.Duration(seconds) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	} else if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	} else {
		return fmt.Sprintf("%ds", s)
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
