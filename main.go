package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

func main() {
	// Get the output format and limit from command-line arguments
	outputFormat := flag.String("output", "table", "Output format: 'table' or 'csv'")
	limit := flag.Int("limit", -1, "Limit the number of projects to process (negative means no limit)")
	flag.Parse()

	// Get project IDs using `gcloud projects list`
	cmd := exec.Command("gcloud", "projects", "list", "--format=value(projectId)")
	output, err := cmd.Output()
	if err != nil {
		log.Fatalf("Failed to list projects: %v", err)
	}

	// Split the output into individual project IDs
	projectIDs := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Apply the limit to the number of project IDs to process
	if *limit >= 0 && *limit < len(projectIDs) {
		projectIDs = projectIDs[:*limit]
	}

	ctx := context.Background()

	// Use default application credentials
	credentials, err := google.FindDefaultCredentials(ctx, sqladmin.SqlserviceAdminScope)
	if err != nil {
		log.Fatalf("Failed to find default credentials: %v", err)
	}

	client, err := sqladmin.NewService(ctx, option.WithTokenSource(credentials.TokenSource))
	if err != nil {
		log.Fatalf("Failed to create Cloud SQL Admin client: %v", err)
	}

	// Prepare results storage
	var results [][]string

	// Initialize progress bar
	bar := progressbar.NewOptions(len(projectIDs),
		progressbar.OptionSetDescription("Processing..."),
		progressbar.OptionSetRenderBlankState(true),
	)

	// Iterate over project IDs and list Cloud SQL instances for each project
	for _, projectID := range projectIDs {
		bar.Describe(fmt.Sprintf("Processing project: %s", projectID))
		time.Sleep(100 * time.Millisecond) // Just to make sure progress bar updates
		req := client.Instances.List(projectID)
		if err := req.Pages(ctx, func(page *sqladmin.InstancesListResponse) error {
			for _, instance := range page.Items {
				results = append(results, []string{projectID, instance.Name, instance.DatabaseVersion})
			}
			return nil
		}); err != nil {
			log.Printf("Failed to list instances for project %s: %v", projectID, err)
		}
		bar.Add(1)
	}

	// Get current month and year
	currentTime := time.Now()
	fileName := fmt.Sprintf("cloudsql_version_%s.csv", currentTime.Format("January2006"))

	// Output results to CSV if specified
	if *outputFormat == "csv" {
		f, err := os.Create(fileName)
		if err != nil {
			log.Fatalf("Failed to create CSV file: %v", err)
		}
		defer f.Close()

		writer := csv.NewWriter(f)
		defer writer.Flush()

		// Write header
		writer.Write([]string{"Project ID", "Instance", "Database Version"})

		// Write rows
		for _, row := range results {
			writer.Write(row)
		}

		fmt.Printf("Results written to %s\n", fileName)
	} else {
		// Print the results in a table
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"Project ID", "Instance", "Database Version"})
		for _, v := range results {
			t.AppendRow(table.Row{v[0], v[1], v[2]})
		}
		t.Render()
	}
}
