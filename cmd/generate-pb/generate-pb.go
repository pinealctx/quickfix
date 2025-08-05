package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
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
)

func usage() {
	_, _ = fmt.Fprintf(os.Stderr, "usage: %v [flags] <path to data dictionary> ... \n", os.Args[0])
	_, _ = fmt.Fprintf(os.Stderr, "\nRequired flags:\n")
	_, _ = fmt.Fprintf(os.Stderr, "  -pb_go_pkg string\n        Go package for generated protobuf files\n")
	_, _ = fmt.Fprintf(os.Stderr, "  -pb_root string\n        Directory for generated proto files\n")
	_, _ = fmt.Fprintf(os.Stderr, "  -go_root string\n        Directory for generated Go files\n")
	_, _ = fmt.Fprintf(os.Stderr, "  -fix_pkg string\n        Root import path for QuickFIX packages\n")
	_, _ = fmt.Fprintf(os.Stderr, "\nExample:\n")
	_, _ = fmt.Fprintf(os.Stderr, "  %v -pb_go_pkg github.com/mycompany/proto -pb_root ./proto -go_root ./internal/proto -fix_pkg github.com/mycompany/quickfix spec/FIX44.xml\n", os.Args[0])
	os.Exit(2)
}

func validateFlags() {
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
		_, _ = fmt.Fprintf(os.Stderr, "Error: Missing required flags: %s\n\n", strings.Join(missing, ", "))
		usage()
	}
}

func getPackageName(fixSpec *datadictionary.DataDictionary) string {
	pkg := strings.ToLower(fixSpec.FIXType) + strconv.Itoa(fixSpec.Major) + strconv.Itoa(fixSpec.Minor)

	if fixSpec.ServicePack != 0 {
		pkg += "sp" + strconv.Itoa(fixSpec.ServicePack)
	}

	return pkg
}

type protoComponent struct {
	Package         string
	FIXPackage      string
	FIXSpec         *datadictionary.DataDictionary
	Name            string
	GoPackagePrefix string
	*datadictionary.MessageDef
}

type enumComponent struct {
	GoPackagePrefix string
	FieldTypes      []*datadictionary.FieldType
}

type messagesComponent struct {
	Package         string
	GoPackagePrefix string
	QuickfixRoot    string
	Messages        []messageInfo
}

type messageInfo struct {
	Name    string
	Package string
	*datadictionary.MessageDef
}

func genProtoEnums() {
	c := enumComponent{
		GoPackagePrefix: *pbGoPkg,
		FieldTypes:      internal.GlobalFieldTypes,
	}
	gen(internal.EnumProtoTemplate, path.Join(*pbRoot, "enums.proto"), c)
}

func genEnumExtensions() {
	c := enumComponent{
		GoPackagePrefix: *pbGoPkg,
		FieldTypes:      internal.GlobalFieldTypes,
	}

	// 确定Go文件的输出目录
	outputDir := *goRoot
	if outputDir == "" {
		outputDir = *pbRoot
	}

	gen(internal.EnumExtensionTemplate, path.Join(outputDir, "enum_extensions.go"), c)
}

func genConversions(specs []*datadictionary.DataDictionary) {
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

	c := messagesComponent{
		Package:         "fixmessages",
		GoPackagePrefix: *pbGoPkg,
		QuickfixRoot:    *fixPkg,
		Messages:        allMessages,
	}

	// 确定Go文件的输出目录
	outputDir := *goRoot
	if outputDir == "" {
		outputDir = *pbRoot
	}

	gen(internal.ConversionTemplate, path.Join(outputDir, "conversions.go"), c)
}

func genAllMessages(specs []*datadictionary.DataDictionary) {
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

	c := messagesComponent{
		Package:         "fixmessages", // 统一的包名
		GoPackagePrefix: *pbGoPkg,
		QuickfixRoot:    *fixPkg,
		Messages:        allMessages,
	}

	gen(internal.AllMessagesProtoTemplate, path.Join(*pbRoot, "messages.proto"), c)
}

func gen(t *template.Template, fileOut string, data interface{}) {
	defer waitGroup.Done()
	writer := new(bytes.Buffer)

	if err := t.Execute(writer, data); err != nil {
		errors <- err
		return
	}

	if err := internal.WriteFile(fileOut, writer.String()); err != nil {
		errors <- err
	}
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if flag.NArg() < 1 {
		usage()
	}

	validateFlags()

	// Create proto output directory if it doesn't exist
	if fi, err := os.Stat(*pbRoot); os.IsNotExist(err) {
		if err := os.MkdirAll(*pbRoot, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	} else if !fi.IsDir() {
		log.Fatalf("%v is not a directory", *pbRoot)
	}

	// Create Go output directory if it's different from proto directory
	goOutputDir := *goRoot
	if goOutputDir == "" {
		goOutputDir = *pbRoot
	}
	if goOutputDir != *pbRoot {
		if fi, err := os.Stat(goOutputDir); os.IsNotExist(err) {
			if err := os.MkdirAll(goOutputDir, os.ModePerm); err != nil {
				log.Fatal(err)
			}
		} else if !fi.IsDir() {
			log.Fatalf("%v is not a directory", goOutputDir)
		}
	}

	args := flag.Args()
	if len(args) == 1 {
		dictpath := args[0]
		if strings.Contains(dictpath, "FIX50SP1") {
			args = append(args, strings.Replace(dictpath, "FIX50SP1", "FIXT11", -1))
		} else if strings.Contains(dictpath, "FIX50SP2") {
			args = append(args, strings.Replace(dictpath, "FIX50SP2", "FIXT11", -1))
		} else if strings.Contains(dictpath, "FIX50") {
			args = append(args, strings.Replace(dictpath, "FIX50", "FIXT11", -1))
		}
	}
	var specs []*datadictionary.DataDictionary

	for _, dataDictPath := range args {
		spec, err := datadictionary.Parse(dataDictPath)
		if err != nil {
			log.Fatalf("Error Parsing %v: %v", dataDictPath, err)
		}
		specs = append(specs, spec)
	}

	internal.BuildGlobalFieldTypes(specs)

	// Generate global enums proto file
	waitGroup.Add(1)
	go genProtoEnums()

	// Generate enum extensions for string conversion
	waitGroup.Add(1)
	go genEnumExtensions()

	// Generate a single file for all messages
	waitGroup.Add(1)
	go genAllMessages(specs)

	// Always generate conversion functions
	waitGroup.Add(1)
	go genConversions(specs)

	go func() {
		waitGroup.Wait()
		close(errors)
	}()

	var h internal.ErrorHandler
	for err := range errors {
		h.Handle(err)
	}

	os.Exit(h.ReturnCode)
}
