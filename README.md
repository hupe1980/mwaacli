# mwaacli
![Build Status](https://github.com/hupe1980/mwaacli/workflows/build/badge.svg) 
[![Go Reference](https://pkg.go.dev/badge/github.com/hupe1980/mwaacli.svg)](https://pkg.go.dev/github.com/hupe1980/mwaacli)
[![goreportcard](https://goreportcard.com/badge/github.com/hupe1980/mwaacli)](https://goreportcard.com/report/github.com/hupe1980/mwaacli)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
> **mwaacli** is a command-line interface (CLI) for interacting with Amazon Managed Workflows for Apache Airflow (MWAA).  
It provides an easy way to manage MWAA environments, execute Airflow CLI commands, and open the MWAA web UI.

## 🚀 Features

- **List MWAA environments**: Easily list all your MWAA environments.
- **Get details of a specific MWAA environment**: Retrieve detailed information about a specific MWAA environment.
- **Execute Airflow CLI commands in MWAA**: Run Airflow CLI commands directly within MWAA.
- **Call Airflow Rest API**: Interact with the Airflow Rest API seamlessly.
- **Open the MWAA web UI in a browser**: Quickly open the MWAA web UI in your default browser.
- **Manage Airflow SecurityBackends**: Handle Airflow SecurityBackends efficiently.

## 📦 Installation

### Using Homebrew (MacOS)
```sh
brew tap hupe1980/mwaacli
brew install mwaacli
```

### Using deb/rpm/apk Packages

Download the `.deb`, `.rpm`, or `.apk` package from the [releases page](https://github.com/hupe1980/mwaacli/releases) and install it using the appropriate package manager for your system.

### Manual Installation

Download the pre-compiled binaries from the [releases page](https://github.com/hupe1980/mwaacli/releases) and copy them to your desired location.


## 🛠 Usage

The mwaacli application supports various commands. Use the `--help` flag to see available commands and their descriptions:

```bash
./mwaacli --help
```

## 🤝 Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## 📝 License

This project is licensed under the MIT License. See the LICENSE file for more details.