/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Contributor represents a git contributor with their statistics
type Contributor struct {
	Name      string
	Email     string
	Commits   int
	Additions int
	Deletions int
}

var lastTimeRange string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitwho [file/directory]",
	Short: "Show git contributors statistics for a file or directory",
	Long: `GitWho analyzes git history for a specific file or directory and 
shows statistics about contributors who made changes to it.

For directories, it recursively analyzes all files within that directory.
Results are sorted with the contributors who made the most changes at the top.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		runGitWho(path, lastTimeRange)
	},
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
	// Define the --last/-l flag
	rootCmd.Flags().StringVarP(&lastTimeRange, "last", "l", "", "Time range for statistics (day, week, month, year)")
}

// isGitRepo checks if the current directory is within a git repository
func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// findGitRoot finds the root directory of the git repository
func findGitRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getDateFilter returns a git date filter based on the timeRange
func getDateFilter(timeRange string) string {
	if timeRange == "" {
		return ""
	}

	now := time.Now()
	var since time.Time

	switch timeRange {
	case "day":
		since = now.AddDate(0, 0, -1)
	case "week":
		since = now.AddDate(0, 0, -7)
	case "month":
		since = now.AddDate(0, -1, 0)
	case "year":
		since = now.AddDate(-1, 0, 0)
	default:
		fmt.Printf("Invalid time range: %s. Using all history.\n", timeRange)
		return ""
	}

	return fmt.Sprintf("--since=%s", since.Format("2006-01-02"))
}

// runGitWho runs the git analysis for a file or directory
func runGitWho(path string, timeRange string) {
	// Check if we're in a git repo
	if !isGitRepo() {
		fmt.Println("Error: Not in a git repository")
		os.Exit(1)
	}

	gitRoot, err := findGitRoot()
	if err != nil {
		fmt.Println("Error finding git root:", err)
		os.Exit(1)
	}

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Printf("Error resolving path %s: %v\n", path, err)
		os.Exit(1)
	}

	// Get relative path from git root
	relPath, err := filepath.Rel(gitRoot, absPath)
	if err != nil {
		fmt.Printf("Error getting relative path from git root: %v\n", err)
		os.Exit(1)
	}

	// Check if path exists
	_, err = os.Stat(absPath)
	if os.IsNotExist(err) {
		fmt.Printf("Error: Path %s does not exist\n", path)
		os.Exit(1)
	}

	// Prepare git log command
	dateFilter := getDateFilter(timeRange)
	args := []string{
		"log",
		"--format=%an|%ae",
		"--numstat",
	}

	if dateFilter != "" {
		args = append(args, dateFilter)
	}

	// Add path argument
	args = append(args, "--", relPath)

	// Execute git log command
	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error executing git log: %v\n", err)
		os.Exit(1)
	}

	// Parse the output and collect contributor statistics
	stats := make(map[string]*Contributor)
	lines := strings.Split(out.String(), "\n")

	currentUser := ""
	currentEmail := ""

	for _, line := range lines {
		if strings.Contains(line, "|") {
			// This is a username|email line
			parts := strings.Split(line, "|")
			if len(parts) == 2 {
				currentUser = parts[0]
				currentEmail = parts[1]
			}
		} else if len(line) > 0 && currentUser != "" && !strings.HasPrefix(line, "commit") {
			// This is a stats line with additions and deletions
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				// Skip binary files
				if parts[0] == "-" && parts[1] == "-" {
					continue
				}

				// Parse additions and deletions
				var additions, deletions int
				fmt.Sscanf(parts[0], "%d", &additions)
				fmt.Sscanf(parts[1], "%d", &deletions)

				key := fmt.Sprintf("%s|%s", currentUser, currentEmail)
				contributor, exists := stats[key]
				if !exists {
					contributor = &Contributor{
						Name:  currentUser,
						Email: currentEmail,
					}
					stats[key] = contributor
				}

				contributor.Commits++
				contributor.Additions += additions
				contributor.Deletions += deletions
			}
		}
	}

	// Convert map to slice for sorting
	contributors := make([]*Contributor, 0, len(stats))
	for _, contributor := range stats {
		contributors = append(contributors, contributor)
	}

	// Sort contributors by total changes (additions + deletions)
	sort.Slice(contributors, func(i, j int) bool {
		totalI := contributors[i].Additions + contributors[i].Deletions
		totalJ := contributors[j].Additions + contributors[j].Deletions
		return totalI > totalJ
	})

	// Display results
	if len(contributors) == 0 {
		fmt.Println("No changes found for the specified path and time range.")
		return
	}

	fmt.Printf("\nContributor Statistics for %s", path)
	if timeRange != "" {
		fmt.Printf(" (last %s)", timeRange)
	}
	fmt.Println("\n")

	fmt.Printf("%-30s %-30s %10s %10s %10s %10s\n",
		"NAME", "EMAIL", "COMMITS", "ADDED", "DELETED", "TOTAL")
	fmt.Println(strings.Repeat("-", 100))

	for _, contributor := range contributors {
		total := contributor.Additions + contributor.Deletions
		fmt.Printf("%-30s %-30s %10d %10d %10d %10d\n",
			truncateString(contributor.Name, 30),
			truncateString(contributor.Email, 30),
			contributor.Commits,
			contributor.Additions,
			contributor.Deletions,
			total)
	}
}

// truncateString truncates a string to the given length if needed
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}
