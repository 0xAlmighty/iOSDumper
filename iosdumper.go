package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, sourceFile)
	if err != nil {
		return err
	}

	return nil
}

// unzip extracts the contents of the zip file to a directory of the same name
func unzip(zipFile, targetDir string) error {
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer reader.Close()

	infoPlistFound := false // Flag to track if Info.plist is found

	for _, file := range reader.File {
		path := filepath.Join(targetDir, file.Name)

		if strings.HasSuffix(path, "Info.plist") {
			infoPlistFound = true
			color.Green("Info.plist found at: %s", path)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		fileReader, err := file.Open()
		if err != nil {
			return err
		}
		defer fileReader.Close()

		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer targetFile.Close()

		if _, err := io.Copy(targetFile, fileReader); err != nil {
			return err
		}
	}

	if !infoPlistFound {
		color.Red("Info.plist not found within the zip file.")
	}

	return nil
}

// convertPlistToXML converts a binary plist file to XML format using plutil
func convertPlistToXML(plistPath, targetDir string) error {
	// Copy Info.plist to target directory before converting
	targetPlistPath := filepath.Join(targetDir, "Info.plist")
	err := copyFile(plistPath, targetPlistPath)
	if err != nil {
		return fmt.Errorf("error copying Info.plist to target directory: %v", err)
	}

	cmd := exec.Command("plutil", "-convert", "xml1", targetPlistPath)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error converting Info.plist to XML format: %v", err)
	}
	color.Green("Successfully converted %s to XML format.", targetPlistPath)
	return nil
}

// highlightKeysInFile reads the file at the given path and prints its content with specific keys highlighted
func highlightKeysInFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	// Define the keys to highlight and their respective colors
	keysToHighlight := map[string]*color.Color{
		"CFBundleURLSchemes":             color.New(color.FgCyan),
		"CFBundleURLName":                color.New(color.FgGreen),
		"CFBundleTypeRole":               color.New(color.FgYellow),
		"CFBundleURLComponents":          color.New(color.FgMagenta),
		"CFBundleComponentPath":          color.New(color.FgRed),
		"CFBundleURLComponentQueryItems": color.New(color.FgBlue),
	}

	// Compile a regular expression to match any of the keys
	var patternParts []string
	for key := range keysToHighlight {
		patternParts = append(patternParts, regexp.QuoteMeta(key))
	}
	pattern := regexp.MustCompile("(" + strings.Join(patternParts, "|") + ")")

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := pattern.FindStringSubmatch(line)
		if len(matches) > 0 {
			// If the line contains one of the keys, highlight the matching part
			key := matches[0]
			keysToHighlight[key].Println(line)
		} else {
			// Otherwise, print the line without color
			fmt.Println(line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file %s: %v", filePath, err)
	}

	return nil
}

// highlightText searches for substrings and applies color highlighting
func highlightText(input string, searchText string, colorize *color.Color) string {
	var buffer bytes.Buffer
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		if strings.Contains(line, searchText) {
			parts := strings.Split(line, searchText)
			for i, part := range parts {
				if i > 0 {
					buffer.WriteString(colorize.Sprint(searchText))
				}
				buffer.WriteString(part)
			}
			buffer.WriteString("\n")
		} else {
			buffer.WriteString(line + "\n")
		}
	}
	return buffer.String()
}

// runRadare2Command runs `r2 -qc 'izz~PropertyList'` on the specified binary within the .app directory
func runRadare2Command(appDir string) error {
	// Assuming the main binary has the same name as the .app directory
	appName := filepath.Base(appDir)             // Get the directory name
	binaryPath := filepath.Join(appDir, appName) // Construct the path to the binary

	// Remove the .app extension from the binary name
	binaryPath = strings.TrimSuffix(binaryPath, filepath.Ext(binaryPath))

	cmd := exec.Command("r2", "-qc", "izz~PropertyList", binaryPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error running r2 command on %s: %v, output: %s", appName, err, string(output))
	}

	// Process the output to highlight "applinks:" in green
	highlightedOutput := highlightText(string(output), "applinks:", color.New(color.FgGreen))
	fmt.Printf("Results from r2 command on %s:\n%s", appName, highlightedOutput)
	return nil
}

// runStringsAndGrep runs `strings` on the app binary, then filters with `grep`
func runStringsAndGrep(binaryPath string) error {
	// First, execute the strings command and filter with grep
	cmd := exec.Command("sh", "-c", fmt.Sprintf("strings '%s' | grep -E '.*\\/.*'", binaryPath))
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error executing strings and grep command: %v", err)
	}

	// Exclude specific patterns
	excludePatterns := []string{"https://", "/Users/", "/Volumes/", "http://", "BuildRoot/"}
	var filteredLines []string
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		exclude := false
		for _, pattern := range excludePatterns {
			if strings.Contains(line, pattern) {
				exclude = true
				break
			}
		}
		if !exclude {
			filteredLines = append(filteredLines, line)
		}
	}

	// Combine filtered lines back into a single string
	filteredOutput := strings.Join(filteredLines, "\n")

	// Print the colored output
	colorOutput := colorizeOutput(filteredOutput)
	fmt.Println("Filtered strings with slashes:", colorOutput)

	return nil
}

// colorizeOutput applies color only to lines matching the specific format: /something/something
func colorizeOutput(input string) string {
	var buffer bytes.Buffer
	lines := strings.Split(input, "\n")
	colorize := color.New(color.FgGreen) // Use cyan for highlighting
	colorize2 := color.New(color.FgRed)  // Use BiugRed an for highlighting
	// Compile a regex to match the specific pattern: /something/something
	pattern := regexp.MustCompile(`\/[^\/\s]+\/[^\/\s]+`)

	for _, line := range lines {
		if pattern.MatchString(line) {
			// If the line matches the pattern, apply color
			buffer.WriteString(colorize.Sprintln(line))
		} else {
			// Otherwise, print the line without color
			buffer.WriteString(colorize2.Sprintln(line))
		}
	}
	return buffer.String()
}

// displayBanner
func displayBanner() {
	banner := `
                          ;                                                                     
           :              ED.                                                                   
          t#,           . E#Wi        :                                            ,;           
  t      ;##W.         ;W E###G.      Ef                         t               f#i j.         
  Ej    :#L:WE        f#E E#fD#W;     E#t             ..       : ED.           .E#t  EW,        
  E#,  .KG  ,#D     .E#f  E#t t##L    E#t            ,W,     .Et E#K:         i#W,   E##j       
  E#t  EE    ;#f   iWW;   E#t  .E#K,  E#t           t##,    ,W#t E##W;       L#D.    E###D.     
  E#t f#.     t#i L##Lffi E#t    j##f E#t fi       L###,   j###t E#E##t    :K#Wfff;  E#jG#W;    
  E#t :#G     GK tLLG##L  E#t    :E#K:E#t L#j    .E#j##,  G#fE#t E#ti##f   i##WLLLLt E#t t##f   
  E#t  ;#L   LW.   ,W#i   E#t   t##L  E#t L#L   ;WW; ##,:K#i E#t E#t ;##D.  .E#L     E#t  :K#E: 
  E#t   t#f f#:   j#E.    E#t .D#W;   E#tf#E:  j#E.  ##f#W,  E#t E#ELLE##K:   f#E:   E#KDDDD###i
  E#t    f#D#;  .D#j      E#tiW#G.    E###f  .D#L    ###K:   E#t E#L;;;;;;,    ,WW;  E#f,t#Wi,,,
  E#t     G#t  ,WK,       E#K##i      E#K,  :K#t     ##D.    E#t E#t            .D#; E#t  ;#W:  
  E#t      t   EG.        E##D.       EL    ...      #G      ..  E#t              tt DWi   ,KK: 
  ,;.          ,          E#t         :              j                                          
                          L:                                                                      

iOSDumper - Find key information
Version: 1.0.0
	`
	color.Yellow(banner)
}

func displayHelp() {
	title := color.New(color.FgCyan, color.Bold).SprintFunc()
	option := color.New(color.FgYellow).SprintFunc()

	fmt.Printf("%s\n", title("Usage: iosdumper <file.ipa>\n"))
	fmt.Printf("%s\n", option("Options:"))
	fmt.Printf("  %s\t%s\n", option("-h, --help"), "Show this help message and exit.")
}

func main() {
	displayBanner()

	helpFlag := flag.Bool("help", false, "Show help message")
	flag.BoolVar(helpFlag, "h", false, "Show help message (shorthand)")

	flag.Parse()

	if *helpFlag || len(flag.Args()) == 0 {
		displayHelp()
		os.Exit(0)
	}

	filePath := os.Args[1]

	if !strings.HasSuffix(filePath, ".ipa") {
		color.Red("Error: The specified file does not have an '.ipa' extension.")
		os.Exit(1)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		color.Red("Error: The specified file does not exist.")
		os.Exit(1)
	}

	fileDir := strings.TrimSuffix(filePath, filepath.Ext(filePath))
	if err := os.Mkdir(fileDir, 0755); err != nil {
		color.Red("Error creating directory: %v", err)
		os.Exit(1)
	}

	newFilePath := filepath.Join(fileDir, filepath.Base(filePath))
	if err := copyFile(filePath, newFilePath); err != nil {
		color.Red("Error copying file: %v", err)
		os.Exit(1)
	}

	zipFilePath := strings.TrimSuffix(newFilePath, filepath.Ext(newFilePath)) + ".zip"
	if err := os.Rename(newFilePath, zipFilePath); err != nil {
		color.Red("Error changing file extension: %v", err)
		os.Exit(1)
	}

	color.Green("File successfully copied and renamed to: %s", zipFilePath)

	// Unzip the file
	if err := unzip(zipFilePath, fileDir); err != nil {
		color.Red("Error unzipping file: %v", err)
		os.Exit(1)
	}

	// Search and convert Info.plist to XML format
	infoPlistPath := filepath.Join(fileDir, "Payload", "*.app", "Info.plist") // Assuming standard IPA structure
	matches, err := filepath.Glob(infoPlistPath)
	if err != nil || len(matches) == 0 {
		color.Red("Info.plist not found or error searching: %v", err)
		os.Exit(1)
	}

	// Convert the first matched Info.plist to XML format and copy to the initial directory
	if err := convertPlistToXML(matches[0], fileDir); err != nil {
		color.Red("Error converting Info.plist to XML format: %v", err)
		os.Exit(1)
	}

	// Ensure the directory path ends with a separator
	if !strings.HasSuffix(fileDir, string(os.PathSeparator)) {
		fileDir += string(os.PathSeparator)
	}

	// Construct the full path to Info.plist
	plistPath := filepath.Join(fileDir, "Info.plist")

	// Debug: Print the path being used to open the file
	fmt.Println("Attempting to open:", plistPath)

	// Attempt to highlight keys in the Info.plist file
	err = highlightKeysInFile(plistPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	// Assuming standard IPA structure for finding .app directories
	appDirs, err := filepath.Glob(filepath.Join(fileDir, "Payload", "*.app"))
	if err != nil {
		color.Red("Error finding .app directories: %v", err)
		os.Exit(1)
	}
	if len(appDirs) == 0 {
		color.Red("No .app directories found.")
		os.Exit(1)
	}

	// Loop through each .app directory
	for _, appDir := range appDirs {
		// Construct the expected main binary name (same as the .app directory, minus the extension)
		appName := filepath.Base(appDir)                                 // Get the .app directory name
		binaryName := strings.TrimSuffix(appName, filepath.Ext(appName)) // Remove .app extension
		binaryPath := filepath.Join(appDir, binaryName)                  // Assume binary is directly inside .app folder

		// First, run Radare2 command as before
		if err := runRadare2Command(appDir); err != nil {
			color.Red("Error running Radare2 command: %v", err)
			os.Exit(1)
		}

		// Next, run strings and grep on the app binary
		if err := runStringsAndGrep(binaryPath); err != nil {
			color.Red("Error running strings and grep on the binary: %v", err)
			os.Exit(1)
		}
	}

	color.Green("File successfully extracted and Info.plist converted to XML format in: %s", fileDir)
}
