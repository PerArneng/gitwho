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
var repoPath string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "gitwho [file/directory]",
	Short: "Show git contributors statistics for a file or directory",
	Long: `GitWho analyzes git history for a specific file or directory and 
shows statistics about contributors who made changes to it.

For directories, it recursively analyzes all files within that directory.
Results are sorted with the contributors who made the most changes at the top.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := "."
		if len(args) == 1 {
			path = args[0]
		}
		runGitWho(path, lastTimeRange, repoPath)
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
	rootCmd.Flags().StringVarP(&repoPath, "repo", "r", "", "Path to the git repository (defaults to current directory)")
}

// isGitRepo checks if the current directory is within a git repository
func isGitRepo(repoPath string) bool {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	return err == nil
}

// findGitRoot finds the root directory of the git repository
func findGitRoot(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// findRepoForPath determines which Git repository a file or directory belongs to
func findRepoForPath(path string) (string, error) {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("Error resolving path %s: %v", path, err)
	}

	// Check if path exists
	_, err = os.Stat(absPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("Error: Path %s does not exist", path)
	}

	// If path is a file, use its directory
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("Error checking file info: %v", err)
	}
	
	dirPath := absPath
	if !fileInfo.IsDir() {
		dirPath = filepath.Dir(absPath)
	}

	// Walk up the directory tree until we find a .git directory
	currentDir := dirPath
	for {
		// Check if this directory contains a .git directory
		gitDir := filepath.Join(currentDir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return currentDir, nil
		}

		// Move up to parent directory
		parentDir := filepath.Dir(currentDir)
		
		// If we've reached the root directory and still haven't found a .git dir
		if parentDir == currentDir {
			return "", fmt.Errorf("Could not find a Git repository for path: %s", path)
		}
		
		currentDir = parentDir
	}
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
func runGitWho(path string, timeRange string, repoPath string) {
	var effectiveRepoPath string
	var err error

	// If repo path is explicitly specified, use it
	if repoPath != "" {
		effectiveRepoPath = repoPath
	} else {
		// Otherwise, automatically detect the repository for the given path
		effectiveRepoPath, err = findRepoForPath(path)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Printf("Found Git repository: %s\n", effectiveRepoPath)
	}

	// Check if it's a valid git repo
	if !isGitRepo(effectiveRepoPath) {
		fmt.Printf("Error: %s is not a git repository\n", effectiveRepoPath)
		os.Exit(1)
	}

	// Get relative path from git root
	relPath, err := getRelativePath(path, effectiveRepoPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Get git log data
	output, err := executeGitLog(relPath, timeRange, effectiveRepoPath)
	if err != nil {
		fmt.Printf("Error executing git log: %v\n", err)
		os.Exit(1)
	}

	// Parse the output and collect contributor statistics
	contributors := parseGitOutput(output)

	// Display results
	displayResults(contributors, path, timeRange)
}

// getRelativePath gets the relative path from git root for the given path
func getRelativePath(path string, repoPath string) (string, error) {
	gitRoot, err := findGitRoot(repoPath)
	if err != nil {
		return "", fmt.Errorf("Error finding git root: %v", err)
	}

	var targetPath string
	if filepath.IsAbs(path) {
		// If path is absolute, use it directly
		targetPath = path
	} else {
		// If path is relative and starts with repo path, use it as is
		if strings.HasPrefix(path, repoPath) {
			targetPath = path
		} else {
			// Otherwise, join with repo path
			targetPath = filepath.Join(repoPath, path)
		}
	}

	// Get absolute path
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return "", fmt.Errorf("Error resolving path %s: %v", path, err)
	}

	// Check if path exists
	_, err = os.Stat(absPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("Error: Path %s does not exist", path)
	}

	// Get relative path from git root
	relPath, err := filepath.Rel(gitRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("Error getting relative path from git root: %v", err)
	}

	return relPath, nil
}

// executeGitLog runs the git log command and returns its output
func executeGitLog(relPath string, timeRange string, repoPath string) (string, error) {
	// Prepare git log command
	dateFilter := getDateFilter(timeRange)
	args := []string{
		"-C", repoPath,
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

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

// parseGitOutput parses git log output to extract contributor statistics
func parseGitOutput(output string) []*Contributor {
	stats := make(map[string]*Contributor)
	lines := strings.Split(output, "\n")

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
			processStatLine(line, currentUser, currentEmail, stats)
		}
	}

	// Convert map to slice and sort
	return sortContributors(stats)
}

// processStatLine processes a single line of git statistics
func processStatLine(line, currentUser, currentEmail string, stats map[string]*Contributor) {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return
	}

	// Skip binary files
	if parts[0] == "-" && parts[1] == "-" {
		return
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

// sortContributors sorts contributors by total changes (additions + deletions)
func sortContributors(stats map[string]*Contributor) []*Contributor {
	contributors := make([]*Contributor, 0, len(stats))
	for _, contributor := range stats {
		contributors = append(contributors, contributor)
	}

	// Sort contributors by total changes
	sort.Slice(contributors, func(i, j int) bool {
		totalI := contributors[i].Additions + contributors[i].Deletions
		totalJ := contributors[j].Additions + contributors[j].Deletions
		return totalI > totalJ
	})

	return contributors
}

// displayResults shows the contributor statistics
func displayResults(contributors []*Contributor, path string, timeRange string) {
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
