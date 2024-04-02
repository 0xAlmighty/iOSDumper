# iOSDumper 📱🔍

iOSDumper is a tool designed for the static analysis of iOS applications. It automates the extraction of `.ipa` files, analyzes `Info.plist` to convert and highlight key configurations, and scans the app binary for specific strings of interest (applinks), facilitating a deeper understanding of an app's functionalities and security posture.

## Features ✨

- Extracts and analyzes `.ipa` files with ease.
- Converts `Info.plist` from binary to XML format for easier analysis 📑.
- Highlights key information in `Info.plist` for quick insights 🔑.
- Searches app binaries for strings related to property lists, URL schemes, and other patterns of interest. 🔍

## Prerequisites 📋

Before using iOSDumper, ensure your system meets the following requirements:

- **Go Programming Language**: iOSDumper is built with Go. Make sure you have [Go installed](https://golang.org/dl/) on your system.
- **plutil**: This tool is used for converting `Info.plist` files from binary to XML. It's typically available on macOS systems.
- **radare2**: Required for binary analysis. Install it via your system's package manager or download it from the official site.
- **zip/unzip tools**: Ensure you have command-line tools to handle zip files, usually pre-installed on most UNIX-like operating systems.

Check each tool's installation guide to ensure they are correctly set up and accessible from your command line.

## Installation 🚀

Download the release from the release page

Or:

1. Ensure you have Go installed on your machine.
2. Clone the repository:
```
git clone https://github.com/yourusername/iosdumper.git
```
3. Navigate to the cloned directory:
```
cd iosdumper
```
4. Build the tool:
```
go build
```

## Usage 📖

Run `iosdumper` by specifying the path to the `.ipa` file:

bash
```
./iosdumper path/to/app.ipa
```

## Contributing 🤝

Contributions are welcome! If you have a feature request, bug report, or a patch, please feel free to open an issue or submit a pull request. Feel free to reach out almightysec @ pm.me

Happy hacking :)

## License 📄

Distributed under the MIT License. See `LICENSE` for more information.
