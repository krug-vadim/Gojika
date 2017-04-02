package main

import (
	"bytes"
	"fmt"
	"os"
	"io"
	"io/ioutil"
	// "time"
	"html/template"

	"gopkg.in/yaml.v2"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"

	"github.com/miekg/mmark"
	"github.com/jhillyerd/enmime"
)

type PageData struct {
	Author  string
	Title   string
	Date    string
	Content template.HTML
}


// Render a Markdown fragment into a html fragment
func Render(md string) (html string, err error) {
	// not rendering entire page - that will happen using the tmpl html
	page := false
	css := ""
	head := ""

	// set up options
	extensions := 0
	extensions |= mmark.EXTENSION_TABLES
	extensions |= mmark.EXTENSION_FENCED_CODE
	extensions |= mmark.EXTENSION_AUTOLINK
	extensions |= mmark.EXTENSION_SPACE_HEADERS
	extensions |= mmark.EXTENSION_CITATION
	extensions |= mmark.EXTENSION_TITLEBLOCK_TOML
	extensions |= mmark.EXTENSION_HEADER_IDS
	extensions |= mmark.EXTENSION_AUTO_HEADER_IDS
	extensions |= mmark.EXTENSION_UNIQUE_HEADER_IDS
	extensions |= mmark.EXTENSION_FOOTNOTES
	extensions |= mmark.EXTENSION_SHORT_REF
	extensions |= mmark.EXTENSION_INCLUDE
	extensions |= mmark.EXTENSION_PARTS
	extensions |= mmark.EXTENSION_ABBREVIATIONS
	extensions |= mmark.EXTENSION_DEFINITION_LISTS

	var renderer mmark.Renderer
	htmlFlags := 0
	if page {
		htmlFlags |= mmark.HTML_COMPLETE_PAGE
	}
	renderer = mmark.HtmlRenderer(htmlFlags, css, head)

	output := mmark.Parse([]byte(md), renderer, extensions).Bytes()
	return string(output), nil
}

type T struct {
        A string
        B struct {
                RenamedC int   `yaml:"c"`
                D        []int `yaml:",flow"`

       }
   }

func ReadConfig(FileName string) {
	data, err := ioutil.ReadFile(FileName)
	if err != nil {
		fmt.Println("[e] Error open config file:", err)
		return
	}

	t := T{}

	err = yaml.Unmarshal(data, &t)
	if err != nil {
		fmt.Println("error: ", err)
	}
	fmt.Fprintf(os.Stderr,"--- t:\n%v\n\n", t)
}

func main() {
	env, err := enmime.ReadEnvelope(os.Stdin)
	if err != nil {
		fmt.Println("[e] ", err);
		return;
	}

	ReadConfig("config.yml")

	keyFile, err := os.Open("pub.key")
	if err != nil {
		fmt.Println("[e] PubKey Error:", err);
		return;
	}

	keyRing, err := openpgp.ReadArmoredKeyRing(keyFile)
	// armoredKey, err := ioutil.ReadFile("pub.key")
	if err != nil {
		fmt.Println("[e] PubKey Error:", err);
		return;
	}

	fmt.Fprintf(os.Stderr, "From: %v\n", env.GetHeader("From"));

	for i := range env.Attachments {
		fmt.Println(env.Attachments[i].FileName, env.Attachments[i].ContentType);

		f, _ := os.Create(env.Attachments[i].FileName)
		defer f.Close()
		io.Copy(f, env.Attachments[i]);
	}

	blk, _ := clearsign.Decode([]byte(env.Text));
	if blk == nil {
		fmt.Println("[e] No clearsign message!")
		return
	}

	signer, err := openpgp.CheckDetachedSignature(keyRing, bytes.NewReader(blk.Bytes), blk.ArmoredSignature.Body)
	if err != nil {
		fmt.Println("[e] Signer:", err);
		return;
	}

	fmt.Fprintf(os.Stderr,"Signer: ", signer);

	html, err := Render(string(blk.Plaintext));
	if err != nil {
		fmt.Println("[e] ", err);
		return;
	}

	page_template, err := template.ParseFiles("templates/page.html")
	if err != nil {
		fmt.Println("[e] ", err);
		return
	}

	page_data := PageData{
		Author: env.GetHeader("From"),
		Title: env.GetHeader("Subject"),
		Date: env.GetHeader("Date"),
		Content: template.HTML(html)}
	// data["author"] = signer.Identities[0]

	err = page_template.Execute(os.Stdout, page_data)
	if err != nil {
		fmt.Println("[e] ", err);
		return
	}
}