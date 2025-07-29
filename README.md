# PAWS - Pulumi login and stack selector plus AWS Profile Switcher in Go

---

<img src="assets/paws.png" width="200">

paws is a command-line utility that allows you to easily switch between AWS Profiles and if a Pulumi project is detected, it will also allow you to login into the Pulumi state and select the stack you want to work with.

## Table of Contents

- [Installation](#installation)
    - [Hombrew](#homebrew)
    - [Homebrew upgrade](#homebrew-upgrade)
    - [Release Binaries](#release-binaries)
- [Usage](#usage)
    - [Pulumi configuration file](#pulumi-configuration-file)
    - [Switching AWS Profiles](#switching-aws-profiles)
- [Contributing](#contributing)
- [License](#license)

## Installation

### Homebrew
If you are using Homebrew, you can install paws with the following command:

```sh
brew tap lzecca78/paws
brew install paws
```

Add this alias to your shell configuration file (e.g., `.bashrc`, `.zshrc`, etc.):

```sh
alias paws="source _paws"

```
This will rely on [this](./scripts/_paws) script to set up the environment variables and run the `paws` command.

### Homebrew Upgrade
If you already have paws installed via Homebrew, you can upgrade it with the following command:

```sh
brew upgrade --cask lzecca78/paws/paws
```

> ⚠️ **macOS Gatekeeper Warning**
> 
> The Homebrew tap is not signed, so you may need to allow it to run in your macOS security settings. If you encounter an error when trying to run the command, go to `System Preferences > Security & Privacy` and allow the app to run.


### Release Binaries
You can download the latest release binaries from the Releases page on GitHub: [Releases](https://github.com/lzecca78/paws/releases).
Move the binary to a directory in your PATH, such as `/usr/local/bin`, and make it executable:

```sh
chmod +x /path/to/paws
```

## Usage

### Pulumi configuration file

If you want to use the Pulumi functionality, you need to have a `.pulumi_config.yaml` file in your home directory. This file will be use to bind the aws account id to the bucket name where the Pulumi state is stored. The file should look like this:

```yaml
pulumi_projects:
    "0123456789012": "my-pulumi-bucket"
    "1234567890123": "my-other-pulumi-bucket"
    "234567890123": "my-third-pulumi-bucket"
```    

### Switching AWS Profiles

It is possible to shortcut the menu selection by passing the profile name you want to switch to as an argument.

```bash
> paws work
Profile work set.
```

To switch between different profiles files using the menu, use the following command:

```bash
paws
```

This command will display a list of available profiles files in your `~/.aws/config` file or from `AWS_CONFIG_FILE`
if you have that set. It expects for you to have named profiles in your AWS config file. Select the one you want to use.
Furthermore, if a `Pulumi.yaml` file is detected in the current directory, it will also login to the Pulumi state and allow you to select the stack you want to work with.

## Contributing

If you encounter any issues or have suggestions for improvements, please open an issue or create a pull request on [GitHub](https://github.com/lzecca78/awsd).

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.


Inspired by https://github.com/radiusmethod/awsd
