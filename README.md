# Jira Release Manager

🚀 A command-line interface (CLI) tool written in Go to simplify and automate release management on Jira.

This tool interacts with the Jira API to retrieve information about "Fix Versions," allowing you to get a clear overview of upcoming releases and effortlessly generate formatted changelogs.

## ✨ Features

* **Interactive Selection**: An interactive menu to easily choose the Jira version you want to analyze.
* **Hierarchical View**: Displays all tickets in a release in a clean tree structure (Epic > Story/Task > Sub-task).
* **Automatic Changelogs**: Generates formatted changelogs for various platforms, such as **Markdown** (for GitHub, Confluence) and **Microsoft Teams**.
* **Impact Analysis**: Groups tickets by their labels to quickly identify which repositories or components are impacted by a release.
* **Simple Configuration**: Requires only three environment variables to connect to Jira.

## 📦 Installation

Ensure you have **Go (version 1.21 or later)** installed on your system.

You can install the tool directly using `go install`:
```sh
go install [github.com/greds94/jira-release-manager@latest](https://github.com/greds94/jira-release-manager@latest)
```

Alternatively, you can build from the source:
```sh
# Clone the repository
git clone [https://github.com/greds94/jira-release-manager.git](https://github.com/greds94/jira-release-manager.git)

# Navigate into the directory
cd jira-release-manager

# Build the executable
go build
```

## ⚙️ Configuration

Before using the tool, you must configure access to your Jira workspace. Create a `.env` file in the same directory where you run the command (or in a parent directory).

You can start from the `env.example` file:
```env
# Jira Configuration
JIRA_URL=[https://your-domain.atlassian.net](https://your-domain.atlassian.net)
JIRA_USERNAME=your-email@example.com
JIRA_API_TOKEN=your-api-token-here
```

* `JIRA_URL`: The base URL of your Jira Cloud instance.
* `JIRA_USERNAME`: Your Atlassian account email.
* `JIRA_API_TOKEN`: Your API token. You can generate one from your Atlassian account's security settings [here](https://id.atlassian.com/manage-profile/security/api-tokens).

## 🚀 Usage

The basic format for all commands is:
```sh
jira-release-manager <command> --project <PROJECT_KEY> [flags]
```

### `list-versions`

Displays a table of all project versions, their status, and release dates. Useful for getting a high-level overview.

```sh
jira-release-manager list-versions -p PROJ
```

### `next-release`

Shows a detailed hierarchical view of all tickets included in a version. You will be interactively prompted to select which version to analyze.

```sh
jira-release-manager next-release -p PROJ
```

Add the `--detailed` (or `-d`) flag to see more details for each ticket, such as priority and assignee.

```sh
jira-release-manager next-release -p PROJ -d
```

### `changelog`

Generates a formatted changelog for the selected version.

**Example 1: Print to console in Markdown format**
```sh
jira-release-manager changelog -p PROJ
```

**Example 2: Save to a file in Microsoft Teams format**
```sh
jira-release-manager changelog -p PROJ --format teams --output CHANGELOG.md
```

**Available Flags:**
* `--format` (`-f`): Specifies the output format (`markdown`, `teams`). Default: `markdown`.
* `--output` (`-o`): Saves the result to a file instead of printing to the console.
* `--include-subtasks` (`-s`): Also includes sub-tasks in the generated changelog.

### `impacted-repos`

Shows the selected version's tickets grouped by label. This is ideal for understanding which repositories or services are involved in the release.

```sh
jira-release-manager impacted-repos -p PROJ
```

## 🤝 Contributing

If you wish to contribute, please open an issue to discuss your idea or submit a pull request with your changes. All contributions are welcome!

## 📄 License

This project is licensed under the MIT License. See the `LICENSE` file for more details.