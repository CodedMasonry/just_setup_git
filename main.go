package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	gitContext string
	username   string
	email      string
	ignoreSSH  bool
	sshPath    string
	signing    bool
)

var (
	successColor = lipgloss.NewStyle().Foreground(lipgloss.Color("#A8CC8C"))
	warningColor = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#DBAB79"))
	errorColor   = lipgloss.NewStyle().Foreground(lipgloss.Color("#E88388"))
	infoColor    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#71BEF2"))
)

func main() {
	if err := promptForm(); err != nil {
		panic(err)
	}

	if err := setupGit(); err != nil {
		panic(err)
	}

	if !ignoreSSH {
		if err := setupSSH(); err != nil {
			panic(err)
		}
	}

	if signing {
		if err := setupSigning(); err != nil {
			panic(err)
		}
	}
	sshPath = "/home/mason/.ssh/id_ed25519"

	if !ignoreSSH || signing {
		almostDone()
	} else {
		success("Finished!")
	}
}

// CLI rendering for success
func success(msg ...string) {
	fmt.Print(successColor.Render("✓ "))
	fmt.Println(successColor.Render(msg...))
}

// CLI rendering for success
func warning(msg ...string) {
	fmt.Println(warningColor.Render(msg...))
}

// CLI rendering for success
func panic(err error) {
	fmt.Println(errorColor.Render("! Fatal Error !\n"))
	log.Fatal(err)
}

// Ask user questions
func promptForm() (err error) {
	// ssh path inferencing
	currentUser, err := user.Current()
	sshPath = fmt.Sprintf("%s/.ssh/id_ed25519", currentUser.HomeDir)

	form := huh.
		NewForm(
			// Git
			huh.NewGroup(
				// Global or Local git
				huh.NewSelect[string]().
					Title("How do you want to set up git?").
					Value(&gitContext).
					Description("Global refers to across the system, while local refers to a specific repo").
					Options(
						huh.NewOption("Global (Recommended)", "--global"),
						huh.NewOption("Local  (Not Recommended)", "--local"),
					),

				// Username
				huh.NewInput().
					Title("Username to display for commits?").
					Description("Using one associated with your account is recommended").
					Value(&username).
					Validate(func(str string) error {
						if str == "" {
							return errors.New("Username Required")
						}
						return nil
					}),

				// Email
				huh.NewInput().
					Title("Email used for commits?").
					Value(&email).
					Description("For github, it is recommended to use your `no-reply` email. It can be found at `https://github.com/settings/emails`").
					Placeholder("number+user@users.noreply.github.com").
					Validate(func(str string) error {
						if !strings.Contains(str, "@") || !strings.Contains(str, ".") {
							return errors.New("Not valid email address")
						}
						return nil
					}),

				// ask if SSH is wanted
				// Have to flip bool because of how `huh` displays options.
				huh.NewSelect[bool]().
					Title("Create & setup an SSH key?").
					Value(&ignoreSSH).
					Description("SSH is the preferred method for connecting to git servers.").
					Options(
						huh.NewOption("Yes", false),
						huh.NewOption("No", true),
					),

				// file for SSH key
				huh.NewInput().
					Title("File in which to save SSH key?").
					Value(&sshPath),

				// ask if want signing
				huh.NewSelect[bool]().
					Title("Preferred Commit signing?").
					Value(&signing).
					Description("Isn't required, but is good practice to use").
					Options(
						huh.NewOption("None", false),
						huh.NewOption("SSH", true),
					),
			),
		)

	return form.Run()
}

// Setup git
func setupGit() error {
	err := exec.Command("git", "config", gitContext, "user.name", username).Run()
	if err != nil {
		return err
	}
	success("Successfully set git username")

	err = exec.Command("git", "config", gitContext, "user.email", username).Run()
	if err != nil {
		return err
	}
	success("Successfully set git email")

	return nil
}

func generateSSH() error {
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-f", sshPath, "-C", email)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		err := errors.New("Failed while generating SSH key")
		return err
	}
	success("Successfully generated ssh key")

	return nil
}

// Setup SSH if selected
func setupSSH() error {
	err := generateSSH()
	if err != nil {
		return err
	}

	err = exec.Command("ssh-add", sshPath).Run()
	if err != nil {
		warning("Warning Failed to add SSH key using `ssh-add`\nThis can happen if the key is already added. If it isn't, try running `ssh-add %s", sshPath)
	}

	return nil
}

// Setup Signing if selected
func setupSigning() error {
	if ignoreSSH {
		err := generateSSH()
		if err != nil {
			return err
		}
	}

	err := exec.Command("git", "config", gitContext, "gpg.format", "ssh").Run()
	if err != nil {
		return err
	}
	success("Successfully set git signing type")

	err = exec.Command("git", "config", gitContext, "user.signingkey", sshPath).Run()
	if err != nil {
		return err
	}
	success("Successfully set git signing key")

	err = exec.Command("git", "config", gitContext, "commit.gpgsign", "true").Run()
	if err != nil {
		return err
	}
	success("Successfully set git to sign commits by default")

	return nil
}

// Last information lines if ssh was set up
func almostDone() error {
	pub, err := os.ReadFile(sshPath + ".pub")
	if err != nil {
		return err
	}

	lines := []string{
		infoColor.Render("\nAlmost Done!"),
		"If you're using github, navigate to `https://github.com/settings/keys` and press `New SSH key`",
		"If you set up SSH, set `Key type` to `Authentication Key`",
		"If you set up Signing, set `Key type` to `Signing Key`",
		"You have to create seperate keys on github, but can use the same key on your machine\n",
		"Key:",
		infoColor.Render(string(pub)),
		"\nIf you setup SSH, to avoid re-typing ssh key to push, it is recommended to add `ssh-agent` to your command line settings",
		infoColor.Render("eval \"$(ssh-agent -s)\""),
	}
	for _, line := range lines {
		fmt.Println(line)
	}

	return nil
}
