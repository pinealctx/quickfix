package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"

	"github.com/quickfixgo/quickfix/cmd/generate-pb/internal"
	"github.com/quickfixgo/quickfix/datadictionary"
)

var (
	waitGroup sync.WaitGroup
	errors    = make(chan error)

	// Command line flags - all required
	pbGoPkg = flag.String("pb_go_pkg", "", "Go package for generated protobuf files (required)")
	pbRoot  = flag.String("pb_root", "", "Directory for generated proto files (required)")
	goRoot  = flag.String("go_root", "", "Directory for generated Go files (required)")
	fixPkg  = flag.String("fix_pkg", "", "Root import path for QuickFIX packages (required)")

	// Additional configuration flags
	verbose    = flag.Bool("verbose", false, "Enable verbose output")
	dryRun     = flag.Bool("dry-run", false, "Perform dry run without writing files")
	validate   = flag.Bool("validate", true, "Validate generated code (disable for faster generation)")
	packageDoc = flag.String("package-doc", "", "Package documentation comment")
)

// Config holds the validated configuration
type Config struct {
	PbGoPkg    string
	PbRoot     string
	GoRoot     string
	FixPkg     string
	Verbose    bool
	DryRun     bool
	Validate   bool
	PackageDoc string
	InputFiles []string
}

func usage() {
	_, _ = fmt.Fprintf(os.Stderr, "usage: %v [flags] <path to data dictionary> ... \n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "\nRequired flags:\n")
	_, _ = fmt.Fprintf(os.Stderr, "  -pb_go_pkg string\n        Go package for generated protobuf files\n")
	_, _ = fmt.Fprintf(os.Stderr, "  -pb_root string\n        Directory for generated proto files\n")
	_, _ = fmt.Fprintf(os.Stderr, "  -go_root string\n        Directory for generated Go files\n")
	_, _ = fmt.Fprintf(os.Stderr, "  -fix_pkg string\n        Root import path for QuickFIX packages\n")
	_, _ = fmt.Fprintf(os.Stderr, "\nOptional flags:\n")
	_, _ = fmt.Fprintf(os.Stderr, "  -verbose\n        Enable verbose output\n")
	_, _ = fmt.Fprintf(os.Stderr, "  -dry-run\n        Perform dry run without writing files\n")
	_, _ = fmt.Fprintf(os.Stderr, "  -validate\n        Validate generated code (default: true)\n")
	_, _ = fmt.Fprintf(os.Stderr, "  -package-doc string\n        Package documentation comment\n")
	_, _ = fmt.Fprintf(os.Stderr, "\nExample:\n")
	_, _ = fmt.Fprintf(os.Stderr, "  %v -pb_go_pkg github.com/mycompany/proto -pb_root ./proto -go_root ./internal/proto -fix_pkg github.com/mycompany/quickfix spec/FIX44.xml\n", os.Args[0])
	os.Exit(2)
}

func validateConfig() (*Config, error) {
	var missing []string

	if *pbGoPkg == "" {
		missing = append(missing, "-pb_go_pkg")
	}
	if *pbRoot == "" {
		missing = append(missing, "-pb_root")
	}
	if *goRoot == "" {
		missing = append(missing, "-go_root")
	}
	if *fixPkg == "" {
		missing = append(missing, "-fix_pkg")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required flags: %s", strings.Join(missing, ", "))
	}

	// Validate input files
	args := flag.Args()
	if len(args) < 1 {
		return nil, fmt.Errorf("at least one data dictionary file is required")
	}

	// Validate file paths exist
	var inputFiles []string
	for _, dataDictPath := range args {
		if _, err := os.Stat(dataDictPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("data dictionary file does not exist: %s", dataDictPath)
		}

		// Convert to absolute path
		absPath, err := filepath.Abs(dataDictPath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", dataDictPath, err)
		}
		inputFiles = append(inputFiles, absPath)
	}

	// Auto-add FIXT11 if needed (same logic as before)
	if len(inputFiles) == 1 {
		dictpath := inputFiles[0]
		if strings.Contains(dictpath, "FIX50SP1") {
			fixtPath := strings.Replace(dictpath, "FIX50SP1", "FIXT11", -1)
			if _, err := os.Stat(fixtPath); err == nil {
				inputFiles = append(inputFiles, fixtPath)
			}
		} else if strings.Contains(dictpath, "FIX50SP2") {
			fixtPath := strings.Replace(dictpath, "FIX50SP2", "FIXT11", -1)
			if _, err := os.Stat(fixtPath); err == nil {
				inputFiles = append(inputFiles, fixtPath)
			}
		} else if strings.Contains(dictpath, "FIX50") {
			fixtPath := strings.Replace(dictpath, "FIX50", "FIXT11", -1)
			if _, err := os.Stat(fixtPath); err == nil {
				inputFiles = append(inputFiles, fixtPath)
			}
		}
	}

	// Validate package name format
	if !isValidGoPackage(*pbGoPkg) {
		return nil, fmt.Errorf("invalid Go package name: %s", *pbGoPkg)
	}

	return &Config{
		PbGoPkg:    *pbGoPkg,
		PbRoot:     *pbRoot,
		GoRoot:     *goRoot,
		FixPkg:     *fixPkg,
		Verbose:    *verbose,
		DryRun:     *dryRun,
		Validate:   *validate,
		PackageDoc: *packageDoc,
		InputFiles: inputFiles,
	}, nil
}

func isValidGoPackage(pkg string) bool {
	// Basic validation for Go package path
	if pkg == "" {
		return false
	}

	// Check for valid characters and structure
	parts := strings.Split(pkg, "/")
	for _, part := range parts {
		if part == "" || strings.Contains(part, " ") {
			return false
		}
	}

	return true
}

func createDirectories(config *Config) error {
	// Create proto output directory if it doesn't exist
	if err := createDirIfNotExists(config.PbRoot, "proto"); err != nil {
		return err
	}

	// Create Go output directory if it's different from proto directory
	goOutputDir := config.GoRoot
	if goOutputDir == "" {
		goOutputDir = config.PbRoot
	}
	if goOutputDir != config.PbRoot {
		if err := createDirIfNotExists(goOutputDir, "Go"); err != nil {
			return err
		}
	}

	return nil
}

func createDirIfNotExists(dirPath, description string) error {
	if fi, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create %s directory %s: %w", description, dirPath, err)
		}
		if *verbose {
			log.Printf("Created %s directory: %s", description, dirPath)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check %s directory %s: %w", description, dirPath, err)
	} else if !fi.IsDir() {
		return fmt.Errorf("%s path %s exists but is not a directory", description, dirPath)
	}
	return nil
}

func parseDataDictionaries(config *Config) ([]*datadictionary.DataDictionary, error) {
	var specs []*datadictionary.DataDictionary

	for _, dataDictPath := range config.InputFiles {
		if config.Verbose {
			log.Printf("Parsing data dictionary: %s", dataDictPath)
		}

		spec, err := datadictionary.Parse(dataDictPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", dataDictPath, err)
		}

		if config.Verbose {
			log.Printf("Successfully parsed %s (FIX %s.%d.%d)",
				dataDictPath, spec.FIXType, spec.Major, spec.Minor)
		}

		specs = append(specs, spec)
	}

	return specs, nil
}

func getPackageName(fixSpec *datadictionary.DataDictionary) string {
	pkg := strings.ToLower(fixSpec.FIXType) + strconv.Itoa(fixSpec.Major) + strconv.Itoa(fixSpec.Minor)

	if fixSpec.ServicePack != 0 {
		pkg += "sp" + strconv.Itoa(fixSpec.ServicePack)
	}

	return pkg
}

type messagesComponent struct {
	GoPackagePrefix string
	QuickfixRoot    string
	Messages        []messageInfo
}

type messageInfo struct {
	Name    string
	Package string
	*datadictionary.MessageDef
}

func genAllMessages(specs []*datadictionary.DataDictionary, config *Config) {
	var allMessages []messageInfo

	for _, spec := range specs {
		pkg := getPackageName(spec)

		// 处理普通的messages
		for _, msg := range spec.Messages {
			allMessages = append(allMessages, messageInfo{
				Name:       msg.Name,
				Package:    pkg,
				MessageDef: msg,
			})
		}

		// 处理components，将它们也作为messages
		for _, comp := range spec.ComponentTypes {
			// 为component创建一个正确的MessageDef包装器
			componentMsg := datadictionary.NewMessageDef(comp.Name(), "", comp.Parts())

			allMessages = append(allMessages, messageInfo{
				Name:       comp.Name(),
				Package:    pkg,
				MessageDef: componentMsg,
			})
		}
	}

	// 对消息进行排序以保证生成顺序一致
	sort.Slice(allMessages, func(i, j int) bool {
		// 首先按包名排序
		if allMessages[i].Package != allMessages[j].Package {
			return allMessages[i].Package < allMessages[j].Package
		}
		// 然后按消息名排序
		return allMessages[i].Name < allMessages[j].Name
	})

	if config.Verbose {
		log.Printf("Sorted %d messages for consistent generation order", len(allMessages))
	}

	c := messagesComponent{
		GoPackagePrefix: *pbGoPkg,
		QuickfixRoot:    *fixPkg,
		Messages:        allMessages,
	}

	gen(internal.AllMessagesProtoTemplate, path.Join(*pbRoot, "fix.g.proto"), c, config)
}

func gen(t *template.Template, fileOut string, data interface{}, config *Config) {
	defer waitGroup.Done()

	if config.Verbose {
		log.Printf("Generating file: %s", fileOut)
	}

	writer := new(bytes.Buffer)

	if err := t.Execute(writer, data); err != nil {
		errors <- fmt.Errorf("template execution failed for %s: %w", fileOut, err)
		return
	}

	if config.DryRun {
		if config.Verbose {
			log.Printf("DRY RUN: Would write %d bytes to %s", writer.Len(), fileOut)
		}
		return
	}

	if err := internal.WriteFile(fileOut, writer.String()); err != nil {
		errors <- fmt.Errorf("failed to write %s: %w", fileOut, err)
		return
	}

	if config.Verbose {
		log.Printf("Successfully wrote %s (%d bytes)", fileOut, writer.Len())
	}
}

func genEnumHelpers(config *Config) {
	defer waitGroup.Done()

	if config.Verbose {
		log.Printf("Generating enum helper functions...")
	}

	// 直接生成enum helpers，不使用gen函数以避免重复的waitGroup.Done()
	c := messagesComponent{
		GoPackagePrefix: *pbGoPkg,
		QuickfixRoot:    *fixPkg,
		Messages:        []messageInfo{}, // 这里不需要messages，只需要enum
	}

	enumHelpersFile := path.Join(config.GoRoot, "enum_helpers.go")

	if config.Verbose {
		log.Printf("Generating file: %s", enumHelpersFile)
	}

	writer := new(bytes.Buffer)

	if err := internal.EnumHelpersGoTemplate.Execute(writer, c); err != nil {
		errors <- fmt.Errorf("template execution failed for %s: %w", enumHelpersFile, err)
		return
	}

	if config.DryRun {
		if config.Verbose {
			log.Printf("DRY RUN: Would write %d bytes to %s", writer.Len(), enumHelpersFile)
		}
		return
	}

	if err := internal.WriteFile(enumHelpersFile, writer.String()); err != nil {
		errors <- fmt.Errorf("failed to write %s: %w", enumHelpersFile, err)
		return
	}

	if config.Verbose {
		log.Printf("Successfully wrote %s (%d bytes)", enumHelpersFile, writer.Len())
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()

	// Validate configuration
	config, err := validateConfig()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	if config.Verbose {
		log.Printf("Starting generation with config: %+v", config)
	}

	// Create directories
	if err := createDirectories(config); err != nil {
		log.Fatalf("Directory creation error: %v", err)
	}

	// Parse data dictionaries
	specs, err := parseDataDictionaries(config)
	if err != nil {
		log.Fatalf("Data dictionary parsing error: %v", err)
	}

	if config.Verbose {
		log.Printf("Building global field types from %d specifications", len(specs))
	}

	internal.BuildGlobalFieldTypes(specs)

	// Initialize enum registry with parsed specifications
	if config.Verbose {
		log.Printf("Initializing enum registry...")
	}
	internal.InitializeEnumRegistry(specs)

	// Generate files
	if config.Verbose {
		log.Printf("Generating protobuf files...")
	}

	// Generate a single file for all messages (includes enum definitions)
	waitGroup.Add(1)
	go func() {
		genAllMessages(specs, config)
	}()

	// Generate enum helper functions
	waitGroup.Add(1)
	go func() {
		genEnumHelpers(config)
	}()

	go func() {
		waitGroup.Wait()
		close(errors)
	}()

	var h internal.ErrorHandler
	for err := range errors {
		h.Handle(err)
	}

	// Exit with error if any occurred
	if h.Err() != nil {
		if config.Verbose {
			log.Printf("Generation completed with errors")
		}
		os.Exit(1)
	}

	if config.Verbose {
		log.Printf("Generation completed successfully")
	}
}
