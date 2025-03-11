package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

const repoPath = "."

type model struct {
	menuIndex int
	output    string
}

var menuOptions = []string{
	"Initialize Repository",
	"Add Remote",
	"Stage Changes",
	"Commit Changes",
	"Push to Remote",
	"Pull from Remote",
	"Show Status",
	"Show Branch",
	"Show Log",
	"Merge Branch",
	"View Diff",
}

var theme = lipgloss.NewStyle().Foreground(lipgloss.Color("#cdd6f4"))

func executeGitCommand(args ...string) string {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %s\n%s", err, out)
	}
	return string(out)
}

func getUnstagedFiles() []string {
	cmd := exec.Command("git", "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return []string{"Error fetching status"}
	}

	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) > 3 {
			files = append(files, line[3:])
		}
	}
	return files
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "up":
			if m.menuIndex > 0 {
				m.menuIndex--
			}
		case "down":
			if m.menuIndex < len(menuOptions)-1 {
				m.menuIndex++
			}
		case "enter":
			m.output = handleGitAction(menuOptions[m.menuIndex])
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString("Gitify - Manage Git Repos\n\n")

	for i, option := range menuOptions {
		if i == m.menuIndex {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#fab387")).Render("➡ " + option) + "\n")
		} else {
			b.WriteString("  " + option + "\n")
		}
	}

	b.WriteString("\n" + theme.Render(m.output))
	b.WriteString("\nPress 'q' to exit.")
	return b.String()
}

func handleGitAction(action string) string {
	switch action {
		case "Initialize Repository":
			return executeGitCommand("init")

		case "Add Remote":
			var remoteName string
			var remoteURL string

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
							Title("Remote Name").
							Placeholder("origin").
							Value(&remoteName).
							Validate(func(s string) error {
									if s == "" {
											return errors.New("remote name cannot be empty")
									}
									return nil
							}),
					huh.NewInput().
							Title("Remote URL").
							Placeholder("https://github.com/username/repo.git").
							Value(&remoteURL).
							Validate(func(s string) error {
									if s == "" {
											return errors.New("remote URL cannot be empty")
									}
									return nil
							}),
				),
			).WithTheme(huh.ThemeCatppuccin())

			if err := form.Run(); err == nil {
				if remoteName == "" || remoteURL == "" {
					return "Remote name and URL cannot be empty."
				}

				addResult := executeGitCommand("remote", "add", remoteName, remoteURL)

				if strings.Contains(addResult, "error") || strings.Contains(addResult, "fatal") {
					return fmt.Sprintf("Failed to add remote: %s", addResult)
				}

				remotes := executeGitCommand("remote", "-v")

				return fmt.Sprintf("Remote added successfully: %s -> %s\n\nAll remotes:\n%s", 
					remoteName, remoteURL, remotes)
			}

		case "Stage Changes":
			files := getUnstagedFiles()
			if len(files) == 0 {
				return "No unstaged changes found."
			}

			var selectedOptions []string

			fileOptions := make([]huh.Option[string], 0, len(files)+2)

			for _, file := range files {
				fileOptions = append(fileOptions, huh.NewOption(file, file))
			}

			selection := huh.NewMultiSelect[string]().
				Title("Select files to stage").
				Options(fileOptions...).
				Value(&selectedOptions)

			form := huh.NewForm(huh.NewGroup(selection)).WithTheme(huh.ThemeCatppuccin())
			if err := form.Run(); err == nil {
				if len(selectedOptions) == 0 {
					return "No files selected."
				}

				for _, selected := range selectedOptions {
					if selected == "select_all" {
						return executeGitCommand(append([]string{"add"}, files...)...)
					}
					if selected == "deselect_all" {
						return "No files staged."
					}
				}

				return executeGitCommand(append([]string{"add"}, selectedOptions...)...)
			}

		case "Commit Changes":
			inputField := huh.NewInput().Title("Commit Message").CharLimit(100)

			form := huh.NewForm(
				huh.NewGroup(
						inputField,
				),
			).WithTheme(huh.ThemeCatppuccin())

			if err := form.Run(); err != nil {
				fmt.Println("Error running form:", err)
				return "Error: Failed to run form"
			}

			commitMessageAny := inputField.GetValue()
			commitMessage, ok := commitMessageAny.(string)
			if !ok {
				fmt.Println("Error: Commit message is not a string")
				return "Error: Invalid commit message"
			}

			fmt.Printf("Debug: Commit message entered by user: '%s'\n", commitMessage)

			return executeGitCommand("commit", "-am", commitMessage)

		case "Push to Remote":
			currentBranch := strings.TrimSpace(executeGitCommand("rev-parse", "--abbrev-ref", "HEAD"))
			if currentBranch == "" || strings.Contains(currentBranch, "fatal") {
				return "Failed to get current branch."
			}

			remotesOutput := executeGitCommand("remote")
			if remotesOutput == "" {
				return "No remotes found. Add a remote first."
			}

			remotes := strings.Split(strings.TrimSpace(remotesOutput), "\n")

			var selectedRemote string
			var branchName string

			remoteOptions := make([]huh.Option[string], 0, len(remotes))
			for _, remote := range remotes {
				remoteOptions = append(remoteOptions, huh.NewOption(remote, remote))
			}

			var setUpstream bool

			form := huh.NewForm(
				huh.NewGroup(
					huh.NewSelect[string]().
							Title("Remote").
							Options(remoteOptions...).
							Value(&selectedRemote),
					huh.NewInput().
							Title("Branch").
							Placeholder(currentBranch).
							Value(&branchName),
					huh.NewConfirm().
							Title("Set as upstream?").
							Value(&setUpstream),
				),
			).WithTheme(huh.ThemeCatppuccin())

			if err := form.Run(); err == nil {
				if branchName == "" {
					branchName = currentBranch
				}
				
				args := []string{"push"}
				
				if setUpstream {
					args = append(args, "--set-upstream")
				}
				
				args = append(args, selectedRemote, branchName)
				
				result := executeGitCommand(args...)
				
				if strings.Contains(result, "error") || strings.Contains(result, "fatal") {
					return fmt.Sprintf("Push failed: %s", result)
				}
				
				return fmt.Sprintf("Successfully pushed to %s/%s\n\n%s", 
					selectedRemote, branchName, result)
			}

		case "Pull from Remote":
			return executeGitCommand("pull")

		case "Show Status":
			return executeGitCommand("status")

		case "Show Branch":
			return executeGitCommand("branch")

		case "Show Log":
			return executeGitCommand("log", "--oneline")

		case "Merge Branch":
			return executeGitCommand("merge", "main")

		case "View Diff":
			return executeGitCommand("diff")
	}

	return "Unknown action."
}

func main() {
	p := tea.NewProgram(model{})
	_, err := p.Run()
	if err != nil {
		fmt.Println("Error starting application:", err)
	}
}