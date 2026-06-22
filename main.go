package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/bufbuild/protocompile"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ProtoMethod struct {
	Name     string
	Request  string
	Response string
}

type ProtoField struct {
	Name string
	Type string
}

type ProtoMessage struct {
	Name   string
	Fields []ProtoField
}

const assetsDir = "assets"

var (
	colorWhiteBg   = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	colorSidebarBg = color.RGBA{R: 240, G: 242, B: 245, A: 255}
	colorConsoleBg = color.RGBA{R: 248, G: 249, B: 250, A: 255}
	colorTextBlack = color.RGBA{R: 18, G: 18, B: 18, A: 255}
	colorBorderRed = color.RGBA{R: 220, G: 53, B: 69, A: 255}
)

type ArtisanLightTheme struct{}

func (m *ArtisanLightTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return colorWhiteBg
	case theme.ColorNameInputBackground:
		return colorWhiteBg
	case theme.ColorNameDisabled:
		return colorTextBlack
	default:
		return theme.DefaultTheme().Color(name, theme.VariantLight)
	}
}

func (m *ArtisanLightTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m *ArtisanLightTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m *ArtisanLightTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

func main() {
	os.Setenv("LANG", "en_US.UTF-8")
	os.Setenv("LC_ALL", "en_US.UTF-8")

	myApp := app.New()
	myApp.Settings().SetTheme(&ArtisanLightTheme{})

	w := myApp.NewWindow("gOKurl - gRPC Artisan Client")
	w.Resize(fyne.Size{Width: 1024, Height: 850})
	w.SetFixedSize(false)

	if err := os.MkdirAll(assetsDir, os.ModePerm); err != nil {
		fmt.Println("Error creating assets directory:", err)
	}

	methodListData := []ProtoMethod{}
	messagesRegistry := make(map[string]ProtoMessage)
	formFields := make(map[string]*widget.Entry)
	var localProtos []string

	var currentProtoPath string
	var selectedMethod ProtoMethod
	var methodSelected bool

	serverAddressInput := widget.NewEntry()
	serverAddressInput.SetPlaceHolder("E.g: localhost:50051")

	inputBorderBg := canvas.NewRectangle(colorBorderRed)
	inputBorderBg.SetMinSize(fyne.Size{Width: 0, Height: 42})
	serverAddressContainer := container.NewStack(
		inputBorderBg,
		container.NewPadded(serverAddressInput),
	)

	methodNameLabel := widget.NewLabelWithStyle("Select a method", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	formContainer := container.NewVBox()

	requestOutput := widget.NewMultiLineEntry()
	requestOutput.TextStyle = fyne.TextStyle{Monospace: true}
	requestOutput.SetText("The outgoing JSON payload will be structured here...")
	requestOutput.Disable()

	reqTerminalBg := canvas.NewRectangle(colorConsoleBg)
	reqScroll := container.NewScroll(container.NewStack(reqTerminalBg, container.NewPadded(requestOutput)))
	reqScroll.SetMinSize(fyne.Size{Width: 0, Height: 120})

	responseOutput := widget.NewMultiLineEntry()
	responseOutput.TextStyle = fyne.TextStyle{Monospace: true}
	responseOutput.SetText("The remote server's response will appear here...")
	responseOutput.Disable()

	resTerminalBg := canvas.NewRectangle(colorConsoleBg)
	resScroll := container.NewScroll(container.NewStack(resTerminalBg, container.NewPadded(responseOutput)))
	resScroll.SetMinSize(fyne.Size{Width: 0, Height: 180})

	loaderBar := widget.NewProgressBarInfinite()
	loaderBar.Hide()

	sendBtn := widget.NewButtonWithIcon("Send Request", theme.ConfirmIcon(), nil)
	sendBtn.Importance = widget.HighImportance
	sendBtn.Disable()

	grpcurlBtn := widget.NewButtonWithIcon("", theme.ComputerIcon(), nil)
	grpcurlBtn.Importance = widget.LowImportance
	grpcurlBtn.Disable()

	validateForm := func() {
		address := strings.TrimSpace(serverAddressInput.Text)
		if address == "" {
			inputBorderBg.FillColor = colorBorderRed
			sendBtn.Disable()
			grpcurlBtn.Disable()
		} else {
			inputBorderBg.FillColor = color.Transparent
			if methodSelected {
				sendBtn.Enable()
				grpcurlBtn.Enable()
			}
		}
		inputBorderBg.Refresh()
	}

	serverAddressInput.OnChanged = func(text string) {
		validateForm()
	}

	grpcurlBtn.OnTapped = func() {
		address := serverAddressInput.Text
		if address == "" {
			address = "localhost:50051"
		}

		payloadItems := []string{}
		for fieldName, entry := range formFields {
			payloadItems = append(payloadItems, fmt.Sprintf(`"%s": "%s"`, fieldName, entry.Text))
		}

		jsonPayload := "{" + strings.Join(payloadItems, ",") + "}"
		methodSymbol := selectedMethod.Name

		cmd := fmt.Sprintf("grpcurl -plaintext -d '%s' %s %s", jsonPayload, address, methodSymbol)

		cmdEntry := widget.NewMultiLineEntry()
		cmdEntry.SetText(cmd)
		cmdEntry.Wrapping = fyne.TextWrapWord
		cmdEntry.Disable() // Asegura que el área de texto sea estrictamente Read-Only

		var d dialog.Dialog
		closeBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
			d.Hide()
		})
		closeBtn.Importance = widget.LowImportance

		copyBtn := widget.NewButtonWithIcon("Copy to Clipboard", theme.ContentCopyIcon(), func() {
			w.Clipboard().SetContent(cmd)
		})

		// Envolvemos el Entry en un Scroll y le asignamos un tamaño mínimo explícito
		cmdScroll := container.NewScroll(cmdEntry)
		cmdScroll.SetMinSize(fyne.NewSize(560, 260))

		// Estructura de layout para asegurar la correcta distribución del espacio expandido
		title := widget.NewLabelWithStyle("Generated grpcurl command:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		topBar := container.NewBorder(nil, nil, nil, closeBtn, title)
		content := container.NewBorder( // No dismiss button text
			topBar,
			copyBtn,
			nil,
			nil,
			cmdScroll,
		)

		d = dialog.NewCustomWithoutButtons("Export to grpcurl", content, w)
		d.Resize(fyne.NewSize(600, 420))
		d.Show()
	}

	sendBtn.OnTapped = func() {
		sendBtn.Disable()
		grpcurlBtn.Disable()
		loaderBar.Show()
		loaderBar.Start()

		address := serverAddressInput.Text
		payloadItems := []string{}
		for fieldName, entry := range formFields {
			payloadItems = append(payloadItems, fmt.Sprintf(`"%s": "%s"`, fieldName, entry.Text))
		}

		jsonPayload := "{" + strings.Join(payloadItems, ",") + "}"
		methodSymbol := selectedMethod.Name

		var reqLog strings.Builder
		fmt.Fprintf(&reqLog, "🌍 [TARGET]: %s\n", address)
		fmt.Fprintf(&reqLog, "📬 [METHOD]: %s\n", methodSymbol)
		fmt.Fprintf(&reqLog, "📦 [JSON]:   %s", jsonPayload)
		requestOutput.SetText(reqLog.String())

		responseOutput.SetText("⌛ Connecting via native Go gRPC channel...")

		go func() {
			time.Sleep(200 * time.Millisecond)

			defer func() {
				loaderBar.Stop()
				loaderBar.Hide()
				validateForm()
			}()

			importPath := filepath.Dir(currentProtoPath)
			protoFile := filepath.Base(currentProtoPath)

			compiler := protocompile.Compiler{
				Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
					ImportPaths: []string{importPath, assetsDir},
				}),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
			defer cancel()

			files, err := compiler.Compile(ctx, protoFile)
			if err != nil {
				responseOutput.SetText(fmt.Sprintf("❌ [PROTO PARSE ERROR]: %v", err))
				return
			}

			var targetMethod protoreflect.MethodDescriptor
			parts := strings.Split(methodSymbol, "/")
			if len(parts) == 2 {
				fullServiceName := protoreflect.FullName(parts[0])
				methodName := protoreflect.Name(parts[1])

				for _, fileDesc := range files {
					services := fileDesc.Services()
					for i := 0; i < services.Len(); i++ {
						serviceDesc := services.Get(i)
						if serviceDesc.FullName() == fullServiceName {
							targetMethod = serviceDesc.Methods().ByName(methodName)
							break
						}
					}
				}
			}

			if targetMethod == nil {
				responseOutput.SetText("❌ [ERROR]: Could not find method descriptor in compiled schema.")
				return
			}

			conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				responseOutput.SetText(fmt.Sprintf("❌ [CONNECTION ERROR]:\n%v", err))
				return
			}
			defer conn.Close()

			dynamicRequest := dynamicpb.NewMessage(targetMethod.Input())
			err = protojson.Unmarshal([]byte(jsonPayload), dynamicRequest)
			if err != nil {
				responseOutput.SetText(fmt.Sprintf("❌ [JSON UNMARSHAL ERROR]:\n%v", err))
				return
			}

			dynamicResponse := dynamicpb.NewMessage(targetMethod.Output())
			responseOutput.SetText("⌛ Sending payload over native gRPC channel...")

			grpcMethodPath := fmt.Sprintf("/%s/%s", targetMethod.Parent().FullName(), targetMethod.Name())
			err = conn.Invoke(ctx, grpcMethodPath, dynamicRequest, dynamicResponse)

			var logResult strings.Builder
			if err != nil {
				resTerminalBg.FillColor = colorConsoleBg
				logResult.WriteString("❌ [gRPC SERVER ERROR]:\n")
				logResult.WriteString(err.Error())
				resTerminalBg.Refresh()
			} else {
				logResult.WriteString("🟢 [RESPONSE JSON]:\n")

				jsonBytes, jsonErr := protojson.Marshal(dynamicResponse)
				if jsonErr != nil {
					resTerminalBg.FillColor = colorConsoleBg
					logResult.WriteString(fmt.Sprintf("%v\n", dynamicResponse.String()))
					resTerminalBg.Refresh()
				} else {
					prettyJSON := formatAndNormalizePayloads(jsonBytes)
					logResult.WriteString(prettyJSON)

					rand.Seed(time.Now().UnixNano())
					resTerminalBg.FillColor = color.RGBA{
						R: uint8(235 + rand.Intn(15)),
						G: uint8(245 + rand.Intn(10)),
						B: uint8(235 + rand.Intn(15)),
						A: 255,
					}
					resTerminalBg.Refresh()
				}
			}
			responseOutput.SetText(logResult.String())
		}()
	}

	sidebarList := widget.NewList(
		func() int { return len(methodListData) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(methodListData[id].Name)
		},
	)

	localProtoList := widget.NewList(
		func() int { return len(localProtos) },
		func() fyne.CanvasObject { return widget.NewLabel("template.proto") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(localProtos[id])
		},
	)

	refreshLocalProtosList := func() {
		localProtos = []string{}
		files, err := os.ReadDir(assetsDir)
		if err == nil {
			for _, file := range files {
				if !file.IsDir() && strings.HasSuffix(file.Name(), ".proto") {
					localProtos = append(localProtos, file.Name())
				}
			}
		}
		localProtoList.Refresh()
	}

	loadProtoFromPath := func(path string) {
		file, err := os.Open(path)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		defer file.Close()

		currentProtoPath = path
		methodListData = []ProtoMethod{}
		messagesRegistry = make(map[string]ProtoMessage)

		parseFullProto(file, &methodListData, messagesRegistry)

		sidebarList.Refresh()
		if len(methodListData) > 0 {
			sidebarList.Select(0)
		}
	}

	triggerLoadProto := func() {
		d := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			defer reader.Close()

			path := reader.URI().Path()
			if filepath.Separator == '\\' && strings.HasPrefix(path, "/") {
				path = strings.TrimPrefix(path, "/")
			}

			filename := filepath.Base(path)
			destinationPath := filepath.Join(assetsDir, filename)

			if filepath.Clean(path) != filepath.Clean(destinationPath) {
				out, err := os.Create(destinationPath)
				if err == nil {
					defer out.Close()
					_, _ = io.Copy(out, reader)
				}
				refreshLocalProtosList()
			}

			loadProtoFromPath(path)
		}, w)

		d.SetFilter(storage.NewExtensionFileFilter([]string{".proto"}))
		d.Show()
	}

	menuFile := fyne.NewMenu("Actions",
		fyne.NewMenuItem("Load .proto", triggerLoadProto),
		fyne.NewMenuItem("Exit", func() { myApp.Quit() }),
	)

	menuHelp := fyne.NewMenu("Help",
		fyne.NewMenuItem("Help Documentation", func() {
			dialog.ShowInformation(
				"gOKurl Help",
				"1. Load a .proto file from the File menu or select one from Assets.\n"+
					"2. Choose the gRPC method from the top list in the sidebar.\n"+
					"3. Fill in the server address and the automatically generated parameters.\n"+
					"4. Execute the Request to inspect the results.",
				w,
			)
		}),
		fyne.NewMenuItem("About", func() {
			dialog.ShowCustom(
				"About gOKurl",
				"Close",
				container.NewVBox(
					widget.NewLabelWithStyle("gRPC Testing Client", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
					widget.NewLabelWithStyle("Version 1.4.0", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
					widget.NewSeparator(),
					widget.NewLabel("Clean design with gray sidebar and high-contrast text boxes."),
				),
				w,
			)
		}),
	)

	mainMenu := fyne.NewMainMenu(menuFile, menuHelp)
	w.SetMainMenu(mainMenu)

	sidebarList.OnSelected = func(id widget.ListItemID) {
		selectedMethod = methodListData[id]
		methodNameLabel.SetText("rpc " + selectedMethod.Name + " (" + selectedMethod.Request + ")")

		formContainer.Objects = nil
		formFields = make(map[string]*widget.Entry)

		reqMessage, existe := messagesRegistry[selectedMethod.Request]

		if existe && len(reqMessage.Fields) > 0 {
			form := widget.NewForm()
			for _, field := range reqMessage.Fields {
				input := widget.NewEntry()
				input.SetPlaceHolder(field.Type)
				form.Append(field.Name, input)
				formFields[field.Name] = input
			}
			formContainer.Add(form)
			methodSelected = true
		} else {
			formContainer.Add(widget.NewLabel("This method does not require parameters or the message struct was not found."))
			methodSelected = true
		}

		validateForm()
		formContainer.Refresh()
	}

	localProtoList.OnSelected = func(id widget.ListItemID) {
		targetPath := filepath.Join(assetsDir, localProtos[id])
		loadProtoFromPath(targetPath)
	}

	refreshLocalProtosList()

	methodsContainer := widget.NewCard("DETECTED METHODS", "", sidebarList)
	localFilesContainer := widget.NewCard("AVAILABLE ASSETS (.proto)", "", localProtoList)

	listsSplit := container.NewVSplit(methodsContainer, localFilesContainer)
	listsSplit.Offset = 0.5

	sidebarBgShape := canvas.NewRectangle(colorSidebarBg)
	sizer := canvas.NewRectangle(color.Transparent)
	sizer.SetMinSize(fyne.Size{Width: 350, Height: 800})

	sidebarWrapper := container.NewStack(sizer, sidebarBgShape, listsSplit)

	actionButtons := container.NewBorder(nil, nil, nil, grpcurlBtn, sendBtn)

	controlPanelContent := container.NewVBox(
		widget.NewLabelWithStyle("gRPC Server Address:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		serverAddressContainer,
		widget.NewSeparator(),
		methodNameLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Request Parameters:", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
		formContainer,
		widget.NewSeparator(),
		actionButtons,
	)
	controlCard := widget.NewCard("CONFIGURATION PANEL", "", controlPanelContent)

	logsContent := container.NewVBox(
		widget.NewLabelWithStyle("Client Request Log:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		reqScroll,
		widget.NewSeparator(),
		loaderBar,
		widget.NewLabelWithStyle("Server Response Log:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		resScroll,
	)
	logsCard := widget.NewCard("SYSTEM CONSOLES", "", logsContent)

	rightLayout := container.NewBorder(controlCard, nil, nil, nil, logsCard)
	mainLayout := container.NewBorder(nil, nil, sidebarWrapper, nil, container.NewPadded(rightLayout))

	w.SetContent(mainLayout)
	validateForm()
	w.ShowAndRun()
}

func parseFullProto(r io.Reader, methods *[]ProtoMethod, messages map[string]ProtoMessage) {
	scanner := bufio.NewScanner(r)

	rpcRe := regexp.MustCompile(`rpc\s+([a-zA-Z0-9_\.]+)\s*\(([^)]+)\)\s+returns\s*\(([^)]+)\)`)
	msgStartRe := regexp.MustCompile(`message\s+([a-zA-Z0-9_]+)\s*\{`)
	fieldRe := regexp.MustCompile(`\s*([a-zA-Z0-9_\.]+)\s+([a-zA-Z0-9_]+)\s*=\s*[0-9]+;`)
	packageRe := regexp.MustCompile(`package\s+([a-zA-Z0-9_\.]+);`)

	var currentMessage *ProtoMessage = nil
	var currentService string
	var packageName string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "//") || line == "" {
			continue
		}

		if strings.HasPrefix(line, "package ") {
			if matches := packageRe.FindStringSubmatch(line); len(matches) == 2 {
				packageName = matches[1]
			}
			continue
		}

		if strings.HasPrefix(line, "service ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentService = strings.TrimSpace(parts[1])
				currentService = strings.TrimSuffix(currentService, "{")
			}
			continue
		}

		if strings.HasPrefix(line, "rpc ") {
			matches := rpcRe.FindStringSubmatch(line)
			if len(matches) == 4 {
				methodName := strings.TrimSpace(matches[1])

				if currentService != "" && !strings.Contains(methodName, "/") {
					if packageName != "" {
						methodName = packageName + "." + currentService + "/" + methodName
					} else {
						methodName = currentService + "/" + methodName
					}
				}

				*methods = append(*methods, ProtoMethod{
					Name:     methodName,
					Request:  strings.TrimSpace(matches[2]),
					Response: strings.TrimSpace(matches[3]),
				})
			}
			continue
		}

		if msgStartRe.MatchString(line) {
			matches := msgStartRe.FindStringSubmatch(line)
			name := matches[1]
			currentMessage = &ProtoMessage{Name: name, Fields: []ProtoField{}}
			continue
		}

		if currentMessage != nil {
			if strings.Contains(line, "}") {
				messages[currentMessage.Name] = *currentMessage
				currentMessage = nil
				continue
			}

			if fieldRe.MatchString(line) {
				matches := fieldRe.FindStringSubmatch(line)
				currentMessage.Fields = append(currentMessage.Fields, ProtoField{
					Type: matches[1],
					Name: matches[2],
				})
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ [PROTO PARSER INTERNAL ERROR]: %v\n", err)
	}
}

func formatAndNormalizePayloads(rawJson []byte) string {
	var genericInterface interface{}
	if err := json.Unmarshal(rawJson, &genericInterface); err != nil {
		return string(rawJson)
	}

	cleanTree := jsonRecursiveParser(genericInterface)

	prettyBytes, err := json.MarshalIndent(cleanTree, "", "  ")
	if err != nil {
		return string(rawJson)
	}
	return string(prettyBytes)
}

func jsonRecursiveParser(node interface{}) interface{} {
	switch typedNode := node.(type) {

	case map[string]interface{}:
		targetFields := map[string]bool{"payload": true, "data": true, "body": true, "content": true}

		for key, value := range typedNode {
			if targetFields[key] {
				if strValue, ok := value.(string); ok {
					trimmed := strings.TrimSpace(strValue)
					if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) ||
						(strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")) {

						var innerInterface interface{}
						if err := json.Unmarshal([]byte(trimmed), &innerInterface); err == nil {
							typedNode[key] = jsonRecursiveParser(innerInterface)
							continue
						}
					}
				}
			}
			typedNode[key] = jsonRecursiveParser(value)
		}
		return typedNode

	case []interface{}:
		for i, value := range typedNode {
			typedNode[i] = jsonRecursiveParser(value)
		}
		return typedNode
	}

	return node
}
