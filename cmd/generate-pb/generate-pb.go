package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"unicode"

	"github.com/quickfixgo/quickfix/datadictionary"
)

var (
	waitGroup sync.WaitGroup
	errors    = make(chan error, 10) // Buffered channel to prevent deadlock

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
	genProto   = flag.Bool("gen-proto", true, "Generate Go code from proto files using protoc")
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
	GenProto   bool
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
	_, _ = fmt.Fprintf(os.Stderr, "  -gen-proto\n        Generate Go code from proto files using protoc (default: true)\n")
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
		dictPath := inputFiles[0]
		if strings.Contains(dictPath, "FIX50SP1") {
			fixtPath := strings.Replace(dictPath, "FIX50SP1", "FIXT11", -1)
			if _, err := os.Stat(fixtPath); err == nil {
				inputFiles = append(inputFiles, fixtPath)
			}
		} else if strings.Contains(dictPath, "FIX50SP2") {
			fixtPath := strings.Replace(dictPath, "FIX50SP2", "FIXT11", -1)
			if _, err := os.Stat(fixtPath); err == nil {
				inputFiles = append(inputFiles, fixtPath)
			}
		} else if strings.Contains(dictPath, "FIX50") {
			fixtPath := strings.Replace(dictPath, "FIX50", "FIXT11", -1)
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
		GenProto:   *genProto,
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
	Packages        []string
}

// GetImportedPackages returns the list of packages needed for conversion code
func (c messagesComponent) GetImportedPackages() []string {
	var imports []string

	// Add specific package imports based on message types
	packageMap := make(map[string]bool)
	for _, msg := range c.Messages {
		pkgPath := c.QuickfixRoot + "/" + msg.Package
		if !packageMap[pkgPath] {
			packageMap[pkgPath] = true
			imports = append(imports, pkgPath)
		}
	}

	return imports
}

func (c messagesComponent) GetNonComponentMessages() []messageInfo {
	var nonComponentMessages []messageInfo
	for _, msg := range c.Messages {
		if msg.IsMessage {
			nonComponentMessages = append(nonComponentMessages, msg)
		}
	}
	return nonComponentMessages
}

func (c messagesComponent) GetComponentMessages() []messageInfo {
	var componentMessages []messageInfo
	for _, msg := range c.Messages {
		if !msg.IsMessage {
			componentMessages = append(componentMessages, msg)
		}
	}
	return componentMessages
}

type fieldInfo struct {
	*datadictionary.FieldDef
}

func (f fieldInfo) GoVariableName() string {
	name := f.GoFieldName()
	if len(name) > 0 && unicode.IsUpper(rune(name[0])) {
		name = string(unicode.ToLower(rune(name[0]))) + name[1:]
	}
	return name
}

func (f fieldInfo) GoFieldName() string {
	return toGoFieldName(f.Name())
}

func (f fieldInfo) GetFIXFunctionName() string {
	// Convert field name to FIX function name
	name := f.Name()
	if len(name) > 0 && unicode.IsLower(rune(name[0])) {
		name = string(unicode.ToUpper(rune(name[0]))) + name[1:]
	}
	return "Get" + name
}

func (f fieldInfo) GetProtoFieldName() string {
	// 获取字段名
	name := f.GoFieldName()

	// 将下划线命名转换为驼峰命名
	var result strings.Builder
	upperNext := false
	for _, r := range name {
		if r == '_' {
			upperNext = true
			continue
		}
		if upperNext {
			result.WriteRune(unicode.ToUpper(r))
			upperNext = false
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

func (f fieldInfo) TypeConvert() string {
	fieldName := f.GetProtoFieldName()
	variableName := f.GoVariableName()

	if len(f.Enums) > 0 {
		//return fmt.Sprintf("_ = %s", variableName) // ignore
		return fmt.Sprintf("pbMsg.%s = FIXTo%s[%s]", fieldName, f.Name(), variableName)
	}

	switch f.Type {
	case "STRING", "MULTIPLEVALUESTRING", "MULTIPLESTRINGVALUE", "MULTIPLECHARVALUE":
		return fmt.Sprintf("pbMsg.%s = %s", fieldName, variableName)
	case "CHAR":
		return fmt.Sprintf("pbMsg.%s = string(%s)", fieldName, variableName)
	case "LENGTH":
		return fmt.Sprintf("pbMsg.%s = uint32(%s)", fieldName, variableName)
	case "INT", "SEQNUM", "TAGNUM", "DAYOFMONTH":
		return fmt.Sprintf("pbMsg.%s = int32(%s)", fieldName, variableName)
	case "NUMINGROUP":
		return fmt.Sprintf("_ = %s", variableName) // ignore
	case "AMT", "PERCENTAGE", "PRICE", "QTY", "PRICEOFFSET":
		return fmt.Sprintf(`pbMsg.%s = %s.String()`, fieldName, variableName)
	case "FLOAT":
		//return "float64(" + variableName + ".Float64())"
		return fmt.Sprintf(`pbMsg.%s, _ = %s.Float64()`, fieldName, variableName)
	case "BOOLEAN":
		//return "bool(" + variableName + ")"
		return fmt.Sprintf("pbMsg.%s = bool(%s)", fieldName, variableName)
	case "UTCTIMESTAMP":
		//return variableName + ".Unix()"
		return fmt.Sprintf("pbMsg.%s = %s.Format(\"2006-01-02T15:04:05.999999999Z07:00\")", fieldName, variableName)
	case "UTCDATE", "UTCTIMEONLY", "LOCALMKTDATE", "TZTIMEONLY", "TZTIMESTAMP":
		//return variableName + ".String()"
		return fmt.Sprintf("pbMsg.%s = %s", fieldName, variableName)
	case "DATA", "XMLDATA":
		//return "string(" + variableName + ")"
		return fmt.Sprintf("pbMsg.%s = string(%s)", fieldName, variableName)
	case "CURRENCY", "EXCHANGE", "COUNTRY":
		//return variableName + ".String()"
		return fmt.Sprintf("pbMsg.%s = %s", fieldName, variableName)
	case "MONTHYEAR":
		//return variableName + ".String()"
		return fmt.Sprintf("pbMsg.%s = %s", fieldName, variableName)
	case "TENOR":
		//return variableName + ".String()"
		return fmt.Sprintf("pbMsg.%s = %s", fieldName, variableName)
	default:
		// 对于未知类型，默认转换为字符串
		//return variableName + ".String()"
		return fmt.Sprintf("pbMsg.%s = %s", fieldName, variableName)
	}
}

func (f fieldInfo) ConvertCodes() string {
	b := strings.Builder{}
	b.WriteString(fmt.Sprintf(`
	%s, err := fixMsg.%s()
	if err != nil {
		return nil, fmt.Errorf("failed to get %s from FIX message: %%w", err)
	}
	%s`, f.GoVariableName(), f.GetFIXFunctionName(), f.Name(), f.TypeConvert()))

	return b.String()
}

type messageInfo struct {
	Name    string
	Package string
	*datadictionary.MessageDef
	IsMessage bool
}

func (m *messageInfo) FIXType() string {
	return fmt.Sprintf("%s.%s", strings.ToLower(m.Name), toGoFieldName(m.Name))
}

func (m *messageInfo) GetFields() []fieldInfo {
	fields := getFields(m.MessageDef)
	out := make([]fieldInfo, len(fields))
	for i, f := range fields {
		out[i] = fieldInfo{FieldDef: f}
	}
	return out
}

func genAllMessages(specs []*datadictionary.DataDictionary, config *Config) {
	defer func() {
		if config.Verbose {
			log.Printf("Calling waitGroup.Done() for genAllMessages")
		}
		waitGroup.Done()
	}()

	var packages []string

	var allMessages []messageInfo

	for _, spec := range specs {
		pkg := getPackageName(spec)

		// 处理普通的messages
		for _, msg := range spec.Messages {
			allMessages = append(allMessages, messageInfo{
				Name:       msg.Name,
				Package:    pkg,
				MessageDef: msg,
				IsMessage:  true,
			})
			packages = append(packages, fmt.Sprintf("%s/%s/%s", config.FixPkg, pkg, strings.ToLower(msg.Name)))
		}

		// 处理components，将���们也作为messages
		for _, comp := range spec.ComponentTypes {
			// 为component创建一个正确的MessageDef包装器
			componentMsg := datadictionary.NewMessageDef(comp.Name(), "", comp.Parts())

			allMessages = append(allMessages, messageInfo{
				Name:       comp.Name(),
				Package:    pkg,
				MessageDef: componentMsg,
				IsMessage:  false,
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

	sort.Slice(packages, func(i, j int) bool {
		// 按包名排序
		return packages[i] < packages[j]
	})

	if config.Verbose {
		log.Printf("Sorted %d messages for consistent generation order", len(allMessages))
	}

	c := messagesComponent{
		GoPackagePrefix: *pbGoPkg,
		QuickfixRoot:    *fixPkg,
		Messages:        allMessages,
		Packages:        packages,
	}

	// Generate enum proto file
	genSync(EnumProtoTemplate, path.Join(*pbRoot, "fix.enum.proto"), c, config)

	// Generate component proto file
	genSync(ComponentProtoTemplate, path.Join(*pbRoot, "fix.component.proto"), c, config)

	// Generate message proto file
	genSync(MessageProtoTemplate, path.Join(*pbRoot, "fix.message.proto"), c, config)
}

func genSync(t *template.Template, fileOut string, data interface{}, config *Config) {

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

	if err := WriteFile(fileOut, writer.String()); err != nil {
		errors <- fmt.Errorf("failed to write %s: %w", fileOut, err)
		return
	}

	if config.Verbose {
		log.Printf("Successfully wrote %s (%d bytes)", fileOut, writer.Len())
	}
}

func genEnumConversionFunctions(config *Config) {
	defer func() {
		if config.Verbose {
			log.Printf("Calling waitGroup.Done() for genEnumConversionFunctions")
		}
		waitGroup.Done()
	}()

	if config.Verbose {
		log.Printf("Generating enum helper functions...")
	}

	// 直接生成enum helpers，不使用gen函数以避免重复的waitGroup.Done()
	c := messagesComponent{
		GoPackagePrefix: *pbGoPkg,
		QuickfixRoot:    *fixPkg,
		Messages:        []messageInfo{}, // 这里不需要messages，只需要enum
	}

	enumHelpersFile := path.Join(config.GoRoot, "fix.enum.conversion.go")

	if config.Verbose {
		log.Printf("Generating file: %s", enumHelpersFile)
	}

	writer := new(bytes.Buffer)

	if err := EnumConversionGoTemplate.Execute(writer, c); err != nil {
		errors <- fmt.Errorf("template execution failed for %s: %w", enumHelpersFile, err)
		return
	}

	if config.DryRun {
		if config.Verbose {
			log.Printf("DRY RUN: Would write %d bytes to %s", writer.Len(), enumHelpersFile)
		}
		return
	}

	if err := WriteFile(enumHelpersFile, writer.String()); err != nil {
		errors <- fmt.Errorf("failed to write %s: %w", enumHelpersFile, err)
		return
	}

	if config.Verbose {
		log.Printf("Successfully wrote %s (%d bytes)", enumHelpersFile, writer.Len())
	}
}

func genConversionFunctions(specs []*datadictionary.DataDictionary, config *Config) {
	defer func() {
		if config.Verbose {
			log.Printf("Calling waitGroup.Done() for genConversionFunctions")
		}
		waitGroup.Done()
	}()

	var allMessages []messageInfo
	var packages []string

	for _, spec := range specs {
		pkg := getPackageName(spec)

		// 处理普通的messages
		for _, msg := range spec.Messages {
			allMessages = append(allMessages, messageInfo{
				Name:       msg.Name,
				Package:    pkg,
				MessageDef: msg,
				IsMessage:  true,
			})
			packages = append(packages, fmt.Sprintf("%s/%s/%s", config.FixPkg, pkg, strings.ToLower(msg.Name)))
		}

		// 处理components，将它们也作为messages
		for _, comp := range spec.ComponentTypes {
			// 为component创建一个正确的MessageDef包装器
			componentMsg := datadictionary.NewMessageDef(comp.Name(), "", comp.Parts())

			allMessages = append(allMessages, messageInfo{
				Name:       comp.Name(),
				Package:    pkg,
				MessageDef: componentMsg,
				IsMessage:  false,
			})
		}
	}

	// 对消息进行排序以保证��成顺序一致
	sort.Slice(allMessages, func(i, j int) bool {
		// 首先按包名排序
		if allMessages[i].Package != allMessages[j].Package {
			return allMessages[i].Package < allMessages[j].Package
		}
		// 然后按消息名排序
		return allMessages[i].Name < allMessages[j].Name
	})

	sort.Slice(packages, func(i, j int) bool {
		// 按包名排序
		return packages[i] < packages[j]
	})

	if config.Verbose {
		log.Printf("Generating conversion functions for %d messages", len(allMessages))
		log.Printf("Sorted %d messages for consistent generation order", len(allMessages))
		for i, msg := range allMessages {
			if i < 5 { // Only log first 5 messages to avoid spam
				log.Printf("Message %d: %s (Package: %s)", i, msg.Name, msg.Package)
			}
		}
	}

	c := messagesComponent{
		GoPackagePrefix: *pbGoPkg,
		QuickfixRoot:    *fixPkg,
		Messages:        allMessages,
		Packages:        packages,
	}

	// Generate FIX to Proto conversion functions directly without using gen()
	fixToProtoFile := path.Join(config.GoRoot, "fix.message.conversion.go")

	if config.Verbose {
		log.Printf("Generating file: %s", fixToProtoFile)
	}

	writer := new(bytes.Buffer)

	if err := MessageConversionGoTemplate.Execute(writer, c); err != nil {
		errors <- fmt.Errorf("template execution failed for %s: %w", fixToProtoFile, err)
		return
	}

	if config.Verbose {
		log.Printf("Template executed successfully, generated %d bytes", writer.Len())
	}

	if config.DryRun {
		if config.Verbose {
			log.Printf("DRY RUN: Would write %d bytes to %s", writer.Len(), fixToProtoFile)
		}
		return
	}

	if err := WriteFile(fixToProtoFile, writer.String()); err != nil {
		errors <- fmt.Errorf("failed to write %s: %w", fixToProtoFile, err)
		return
	}

	if config.Verbose {
		log.Printf("Successfully wrote %s (%d bytes)", fixToProtoFile, writer.Len())
		log.Printf("Generated conversion functions")
	}
}

func genProtoGoCode(config *Config) error {
	if !config.GenProto {
		if config.Verbose {
			log.Printf("Skipping protoc code generation (disabled)")
		}
		return nil
	}

	if config.Verbose {
		log.Printf("Generating Go code from proto files using protoc...")
	}

	enumProtoFile := path.Join(config.PbRoot, "fix.enum.proto")
	messageProtoFile := path.Join(config.PbRoot, "fix.message.proto")

	// Check if protoc is available
	if _, err := exec.LookPath("protoc"); err != nil {
		return fmt.Errorf("protoc not found in PATH. Please install Protocol Buffers compiler: %w", err)
	}

	// Build protoc command for both proto files
	args := []string{
		"--proto_path=" + config.PbRoot,
		"--go_out=" + config.GoRoot,
		"--go_opt=paths=source_relative",
		"--go_opt=M" + path.Base(enumProtoFile) + "=" + config.PbGoPkg,
		"--go_opt=M" + path.Base(messageProtoFile) + "=" + config.PbGoPkg,
		enumProtoFile,
		messageProtoFile,
	}

	if config.Verbose {
		log.Printf("Running: protoc %s", strings.Join(args, " "))
	}

	if config.DryRun {
		if config.Verbose {
			log.Printf("DRY RUN: Would run protoc with args: %s", strings.Join(args, " "))
		}
		return nil
	}

	cmd := exec.Command("protoc", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("protoc failed: %w\nOutput: %s", err, string(output))
	}

	if config.Verbose {
		log.Printf("Successfully generated Go code from proto files")
		if len(output) > 0 {
			log.Printf("Protoc output: %s", string(output))
		}
	}

	return nil
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
	if err = createDirectories(config); err != nil {
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

	BuildGlobalFieldTypes(specs)

	// Initialize enum registry with parsed specifications
	if config.Verbose {
		log.Printf("Initializing enum registry...")
	}
	InitializeEnumRegistry(specs)

	// Generate files
	if config.Verbose {
		log.Printf("Generating protobuf files...")
	}

	// Generate proto files (enum and message)
	if config.Verbose {
		log.Printf("Adding 1 to waitGroup for genAllMessages")
	}
	waitGroup.Add(1) // genAllMessages now handles both files synchronously
	go func() {
		genAllMessages(specs, config)
	}()

	// Generate conversion functions
	if config.Verbose {
		log.Printf("Adding 1 to waitGroup for genConversionFunctions")
	}
	waitGroup.Add(1)
	go func() {
		genConversionFunctions(specs, config)
	}()

	// Generate enum helper functions
	if config.Verbose {
		log.Printf("Adding 1 to waitGroup for genEnumConversionFunctions")
	}
	waitGroup.Add(1)
	go func() {
		genEnumConversionFunctions(config)
	}()

	go func() {
		if config.Verbose {
			log.Printf("Starting waitGroup.Wait() to wait for all goroutines to complete")
		}
		waitGroup.Wait()
		if config.Verbose {
			log.Printf("All goroutines completed, closing errors channel")
		}
		close(errors)
	}()

	var h ErrorHandler
	for err := range errors {
		h.Handle(err)
	}

	// Exit with error if any occurred during template generation
	if h.Err() != nil {
		if config.Verbose {
			log.Printf("Generation completed with template errors")
		}
		os.Exit(1)
	}

	// Generate Go code from proto files using protoc
	if err := genProtoGoCode(config); err != nil {
		log.Fatalf("Protoc generation error: %v", err)
	}

	if config.Verbose {
		log.Printf("Generation completed successfully")
	}
}
