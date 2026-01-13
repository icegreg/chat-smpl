package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/icegreg/chat-smpl/cmd/rtuccli/client"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage services",
	Long:  "Monitor and manage microservices",
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List services",
	Long:  "List all monitored services and their status",
	Run: func(cmd *cobra.Command, args []string) {
		apiClient := client.NewAPIClient(serverURL)

		services, err := apiClient.ListServices()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if len(services) == 0 {
			fmt.Println("No services found")
			return
		}

		// Print table
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Service ID", "Name", "Type", "Status", "Health", "Last Check"})
		table.SetBorder(true)
		table.SetRowLine(true)

		for _, svc := range services {
			status := svc.Status
			if svc.Status == "running" {
				status = "● running"
			} else if svc.Status == "error" {
				status = "✗ error"
			} else {
				status = "○ " + svc.Status
			}

			lastCheck := "-"
			if svc.LastCheck != nil {
				lastCheck = formatTimeAgo(*svc.LastCheck)
			}

			table.Append([]string{
				svc.ID,
				svc.Name,
				svc.Type,
				status,
				svc.Health,
				lastCheck,
			})
		}

		fmt.Printf("SERVICES (%d):\n\n", len(services))
		table.Render()
	},
}

var serviceGetCmd = &cobra.Command{
	Use:   "get [service-id]",
	Short: "Get service details",
	Long:  "Get detailed information about a specific service",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiClient := client.NewAPIClient(serverURL)

		svc, err := apiClient.GetService(args[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("SERVICE: %s\n", svc.ID)
		fmt.Printf("Name:        %s\n", svc.Name)
		fmt.Printf("Type:        %s\n", svc.Type)

		status := svc.Status
		if svc.Status == "running" {
			status = "● running"
		} else if svc.Status == "error" {
			status = "✗ error"
		} else {
			status = "○ " + svc.Status
		}
		fmt.Printf("Status:      %s\n", status)
		fmt.Printf("Health:      %s\n", svc.Health)

		if svc.Host != "" {
			fmt.Printf("Host:        %s\n", svc.Host)
		}

		if svc.Port > 0 {
			fmt.Printf("Port:        %d\n", svc.Port)
		}

		if svc.LastCheck != nil {
			fmt.Printf("Last Check:  %s (%s)\n", svc.LastCheck.Format("2006-01-02 15:04:05"), formatTimeAgo(*svc.LastCheck))
		}
	},
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceListCmd)
	serviceCmd.AddCommand(serviceGetCmd)
}

func formatTimeAgo(t time.Time) string {
	diff := time.Since(t)

	if diff < time.Minute {
		return fmt.Sprintf("%ds ago", int(diff.Seconds()))
	} else if diff < time.Hour {
		return fmt.Sprintf("%dm ago", int(diff.Minutes()))
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(diff.Hours()))
	} else {
		return fmt.Sprintf("%dd ago", int(diff.Hours()/24))
	}
}
