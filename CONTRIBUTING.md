# Contributing to FreeCal

First off, thank you for considering contributing to FreeCal! It's people like you that make FreeCal such a great tool.

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code. Please report unacceptable behavior to the project maintainers.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check existing issues as you might find out that you don't need to create one. When you are creating a bug report, please include as many details as possible:

* **Use a clear and descriptive title** for the issue to identify the problem
* **Describe the exact steps which reproduce the problem** in as many details as possible
* **Provide specific examples to demonstrate the steps**
* **Describe the behavior you observed after following the steps** and point out what exactly is the problem with that behavior
* **Explain which behavior you expected to see instead and why**
* **Include screenshots and animated GIFs** which show you following the described steps and clearly demonstrate the problem
* **Include your environment details:**
  * OS and version
  * Go version (`go version`)
  * FreeCal version or commit hash

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

* **Use a clear and descriptive title** for the issue to identify the suggestion
* **Provide a step-by-step description of the suggested enhancement** in as many details as possible
* **Provide specific examples to demonstrate the steps** or provide code snippets
* **Describe the current behavior** and **explain which behavior you expected to see instead** and why
* **Explain why this enhancement would be useful** to most FreeCal users

### Your First Code Contribution

Unsure where to begin contributing? You can start by looking through these issues:

* Issues labeled `good first issue` - issues which should only require a few lines of code
* Issues labeled `help wanted` - issues which should be a bit more involved than `good first issue` issues

### Pull Requests

1. Fork the repo and create your branch from `master`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code follows the existing code style
6. Issue that pull request!

## Development Process

### Setting up your development environment

1. Fork and clone the repository
   ```bash
   git clone https://github.com/your-username/freecal.git
   cd freecal
   ```

2. Install dependencies
   ```bash
   go mod download
   ```

3. Create a new branch for your feature or fix
   ```bash
   git checkout -b feature/your-feature-name
   ```

### Code Style

* Use `gofmt` to format your code
* Follow the [Effective Go](https://golang.org/doc/effective_go.html) guidelines
* Write clear, idiomatic Go code
* Add comments for exported functions and types
* Keep functions small and focused on a single task

### Testing

* Write unit tests for new functionality
* Ensure all tests pass before submitting PR:
  ```bash
  go test ./...
  ```
* Aim for high test coverage for new code

### Commit Messages

* Use the present tense ("Add feature" not "Added feature")
* Use the imperative mood ("Move cursor to..." not "Moves cursor to...")
* Limit the first line to 72 characters or less
* Reference issues and pull requests liberally after the first line
* Consider starting the commit message with an applicable emoji:
  * 🎨 `:art:` when improving the format/structure of the code
  * 🐛 `:bug:` when fixing a bug
  * 🔥 `:fire:` when removing code or files
  * 📝 `:memo:` when writing docs
  * 🚀 `:rocket:` when improving performance
  * ✅ `:white_check_mark:` when adding tests
  * 🔧 `:wrench:` when modifying configuration files

Example:
```
🐛 Fix timezone handling for all-day events

- Correctly parse all-day event dates in calendar timezone
- Add unit tests for timezone edge cases
- Update documentation with timezone examples

Fixes #123
```

## Project Structure

```
freecal/
├── main.go           # Main application entry point
├── go.mod           # Go module definition
├── go.sum           # Go module checksums
├── LICENSE.md       # MIT license
├── README.md        # Project documentation
├── CONTRIBUTING.md  # This file
└── .gitignore      # Git ignore rules
```

## Questions?

Feel free to open an issue with the label `question` if you have any questions about contributing.