package main

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/mdp/qrterminal/v3"
	"golang.design/x/clipboard"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

//go:embed sendText.html
var sendText string

const (
	DataTypeFile = iota
	DataTypeText
)

var data []byte
var dataType int

var sendTextTemplate = template.Must(template.New("sendText").Parse(sendText))

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: send [path]")
		fmt.Println("\tYou can also pass \"paste\" instead of a file path to send from the pasteboard.")
		fmt.Println("\tPass \"-\" to send from stdin. Useful for piping the output of a command (waits for EOF).")
		os.Exit(1)
	}

	if strings.ToLower(os.Args[1]) == "paste" {
		data = clipboard.Read(clipboard.FmtImage)
		dataType = DataTypeFile

		if data == nil {
			data = clipboard.Read(clipboard.FmtText)
			dataType = DataTypeText
		}

		if data == nil {
			fmt.Println("No contents on pasteboard")
			os.Exit(1)
		}
	} else if os.Args[1] == "-" {
		stdinContent, err := io.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}

		data = stdinContent
	} else {
		fileData, err := os.ReadFile(os.Args[1])
		if err != nil {
			panic(err)
		}

		data = fileData
	}

	contentType := http.DetectContentType(data)
	if strings.HasPrefix(contentType, "text/") {
		dataType = DataTypeText
	}

	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	listener, err := ngrok.Listen(ctx,
		config.HTTPEndpoint(),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		return err
	}

	link := listener.URL()

	log.Println("App URL", link)
	qrterminal.GenerateHalfBlock(link, qrterminal.L, os.Stdout)
	return http.Serve(listener, http.HandlerFunc(handler))
}

func handler(w http.ResponseWriter, r *http.Request) {
	if dataType == DataTypeText {
		sendTextTemplate.Execute(w, string(data))
	} else {
		w.Write(data)
	}
}
