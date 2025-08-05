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

	// Command line flags
	goPackagePrefix = flag.String("go_package", "github.com/quickfixgo/quickfix/proto", "Go package prefix for generated protobuf files")
	directory       = flag.String("directory", ".", "Directory to write generated proto files to")
	goDirectory     = flag.String("go_directory", "", "Directory to write generated Go files to (default: same as -directory)")
)

func usage() {
	_, _ = fmt.Fprintf(os.Stderr, "usage: %v [flags] <path to data dictionary> ... \n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(2)
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
	Messages        []messageInfo
}

type messageInfo struct {
	Name    string
	Package string
	*datadictionary.MessageDef
}

func genProtoEnums() {
	c := enumComponent{
		GoPackagePrefix: *goPackagePrefix,
		FieldTypes:      internal.GlobalFieldTypes,
	}
	gen(internal.EnumProtoTemplate, path.Join(*directory, "enums.proto"), c)
}

func genEnumExtensions() {
	c := enumComponent{
		GoPackagePrefix: *goPackagePrefix,
		FieldTypes:      internal.GlobalFieldTypes,
	}

	// 确定Go文件的输出目录
	outputDir := *goDirectory
	if outputDir == "" {
		outputDir = *directory
	}

	gen(internal.EnumExtensionTemplate, path.Join(outputDir, "enum_extensions.go"), c)
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
		GoPackagePrefix: *goPackagePrefix,
		Messages:        allMessages,
	}

	gen(internal.AllMessagesProtoTemplate, path.Join(*directory, "messages.proto"), c)
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

	// Create proto output directory if it doesn't exist
	if fi, err := os.Stat(*directory); os.IsNotExist(err) {
		if err := os.MkdirAll(*directory, os.ModePerm); err != nil {
			log.Fatal(err)
		}
	} else if !fi.IsDir() {
		log.Fatalf("%v is not a directory", *directory)
	}

	// Create Go output directory if it's different from proto directory
	goOutputDir := *goDirectory
	if goOutputDir == "" {
		goOutputDir = *directory
	}
	if goOutputDir != *directory {
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
