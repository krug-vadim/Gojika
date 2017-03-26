package main

import (
	"bytes"
	"fmt"
	"os"
	"io"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"

	"github.com/miekg/mmark"
	"github.com/jhillyerd/enmime"
)

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

func main() {
	env, err := enmime.ReadEnvelope(os.Stdin)
	if err != nil {
		fmt.Println("[e] ", err);
		return;
	}

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

	fmt.Println("Signer: ", signer);

	html, err := Render(string(blk.Plaintext));
	if err != nil {
		fmt.Println("[e] ", err);
		return;
	}
	fmt.Println("md:\n" + html);

	//fmt.Println(entity)
}