/*
 * ALDftr
 * Wiki-Based Knowledge Organizer
 *
 * author: "Mazen A. Melibari"
 * email: "mazen@mazen.ws"
 * license: "MPL"
 * version: "0.1"
 */
package main

import (
	"strings"
	"bytes"
	"regexp"
	"path/filepath"
	"io"
	"io/ioutil"
	"net/http"
	"html/template"
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
	"fmt"
)

var ROOT_PATH = filepath.Dir(os.Args[0])
var __VERSION__ = "0.2"
var STATIC_PATH = filepath.Join(ROOT_PATH, "etc", "static")
var TEMPLATE_PATH = filepath.Join(ROOT_PATH, "etc", "templates")
var METADATA_SEPARATOR = "#####-----|+|-|-|+|-----#####"
var DATA_FOLDER =  filepath.Join(ROOT_PATH, "data") + string(filepath.Separator) // trailing slash is mandatory
var LOCAL_SERVER_PORT = "5000"

type Page struct {
	Name []byte
	Content []byte
	Metadata map[string]string
}

func (p *Page) sanitize_page_name() {
	re := regexp.MustCompile("[^\\p{Arabic}\\w\\d\\:]+")
	p.Name = re.ReplaceAll(p.Name, []byte("-"))
}

func (p *Page) get_file_path() string {
	file_path_parts := strings.Split(string(p.Name), ":")
	file_path_parts[len(file_path_parts)-1] = file_path_parts[len(file_path_parts)-1] + ".txt"

	file_internal_path := filepath.Join(file_path_parts...)
	file_path := filepath.Join(DATA_FOLDER, file_internal_path)
	return file_path
}

func (p *Page) load() error {
	file_path := p.get_file_path()
	content, err := ioutil.ReadFile(file_path)
	if err == nil {
		p.Content = content
		p.parse_file_content()
	}
	return err
}

func (p *Page) save() error {
	file_full_path := p.get_file_path()
	file_dir_path := filepath.Dir(file_full_path)
	os.MkdirAll(file_dir_path, 0600)

	metadata_json, _ := json.Marshal(p.Metadata)
	content := string(p.Content) + "\n" + METADATA_SEPARATOR + "\n" + string(metadata_json)
	return ioutil.WriteFile(file_full_path, []byte(content), 0600)
}

func (p *Page) delete() error {
	file_full_path := p.get_file_path()
	return os.Remove(file_full_path)
}

func (p *Page) parse_file_content() {
	separator := "\n" + METADATA_SEPARATOR + "\n"
	re := regexp.MustCompile(regexp.QuoteMeta(separator) + ".*")

	found_metadata := re.FindAll(p.Content, -1)
	if found_metadata != nil {
		metadata_text := strings.Replace(string(found_metadata[0]), separator, "", 1)
		p.Metadata = make(map[string]string)
		json.Unmarshal([]byte(metadata_text), &p.Metadata)
	}

	p.Content = re.ReplaceAll(p.Content, []byte(""))
}

func get_all_pages(data_folder_path string) []string {
	pages := make([]string, 0, 0)
	folder_separator := string(filepath.Separator)

	visit := func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		page_path := strings.Replace(path, DATA_FOLDER, "", 1)
		page_name := strings.Replace(page_path, folder_separator, ":", -1)

		if page_name != "" {
			pages = append(pages, page_name[:len(page_name)-4])
		}

		return nil

	}

	filepath.Walk(data_folder_path, visit)
	return pages
}

func dftr_format_to_html(txt []byte) []byte {
	// remove \r
	re := regexp.MustCompile(`\r`)
	txt = re.ReplaceAll(txt, []byte(""))

	// local links
	re = regexp.MustCompile(`\[\[([^\]]*)\]\]`)
	txt = re.ReplaceAll(txt, []byte("<a href=\"../view/$1\">$1</a>"))

	// bold
	re = regexp.MustCompile(`\*\*(.*?)\*\*`)
	txt = re.ReplaceAll(txt, []byte("<b>$1</b>"))

	// hr
	re = regexp.MustCompile(`(?m)^\-\-\-\-\-\-*$`)
	txt = re.ReplaceAll(txt, []byte("<hr>"))

	// h6
	re = regexp.MustCompile(`(?m)^\#\#\#\#\#\#(.*)$`)
	txt = re.ReplaceAll(txt, []byte("<h6>$1</h6>"))

	// h5
	re = regexp.MustCompile(`(?m)^\#\#\#\#\#(.*)$`)
	txt = re.ReplaceAll(txt, []byte("<h5>$1</h5>"))

	// h4
	re = regexp.MustCompile(`(?m)^\#\#\#\#(.*)$`)
	txt = re.ReplaceAll(txt, []byte("<h4>$1</h4>"))

	// h3
	re = regexp.MustCompile(`(?m)^\#\#\#(.*)$`)
	txt = re.ReplaceAll(txt, []byte("<h3>$1</h3>"))

	// h2
	re = regexp.MustCompile(`(?m)^\#\#(.*)$`)
	txt = re.ReplaceAll(txt, []byte("<h2>$1</h2>"))

	// h1
	re = regexp.MustCompile(`(?m)^\#(.*)$`)
	txt = re.ReplaceAll(txt, []byte("<h1>$1</h1>"))

	// br
	re = regexp.MustCompile(`\n`)
	txt = re.ReplaceAll(txt, []byte("<br>"))

	return txt
}

func render_template(wr io.Writer, template_name string, data interface{}) {
	var template_results bytes.Buffer
	full_template_path := filepath.Join(TEMPLATE_PATH, template_name + ".html")
	tmpl, _ := template.ParseFiles(full_template_path)
	tmpl.Execute(&template_results, data)

	var page_name string
	value_of_data := data.(map[string]interface{})
	if val, exists := value_of_data["page_name"]; exists {
		page_name = string(val.([]byte))
	} else {
		page_name = ""
	}

	layout_data := map[string]interface {} {
		"Body": template.HTML(template_results.Bytes()),
		"page_name": page_name,
		"__VERSION__": __VERSION__,
	}
	layout_tmpl, _ := template.ParseFiles(filepath.Join(TEMPLATE_PATH, "layout.html"))
	layout_tmpl.Execute(wr, layout_data)
}


func get_page_name_from_url(r *http.Request, action_name string) string {
	page_name := r.URL.Path[len("/" + action_name + "/"):]
	return page_name
}


func index(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "../view/Main", http.StatusFound)
}

func view(w http.ResponseWriter, r *http.Request) {
	page_name := get_page_name_from_url(r, "view")

	page := Page{Name: []byte(page_name)}
	page.sanitize_page_name()

	if page.Name == nil {
		http.Redirect(w, r, "../view/Main", http.StatusNotFound)
		return
	}

	err := page.load()

	if err == nil {
		page.Content = dftr_format_to_html(page.Content)

		// Capitalize first char
		data := map[string]interface {} {
			"page_name": page.Name,
			"page_content": template.HTML(page.Content),
			"page_metadata": page.Metadata,
		}
		render_template(w, "view", data)
	} else {
		http.Redirect(w, r, "../edit/" + string(page.Name), http.StatusFound)
		return
	}
}

func edit(w http.ResponseWriter, r *http.Request) {
	page_name := get_page_name_from_url(r, "edit")

	page := Page{Name: []byte(page_name)}
	page.sanitize_page_name()

	if page.Name == nil {
		http.Redirect(w, r, "../view/Main", http.StatusNotFound)
		return
	}

	page.load()

	metadata, _ := json.Marshal(page.Metadata)

	data := map[string]interface {} {
		"page_name": page.Name,
		"page_content": page.Content,
		"page_metadata": metadata,
	}

	render_template(w, "edit", data)
}

func save(w http.ResponseWriter, r *http.Request) {
	page_name := []byte(r.FormValue("page_name"))
	page_content := []byte(r.FormValue("page_content"))
	page_metadata := []byte(r.FormValue("page_metadata"))

	page := Page{Name: page_name, Content: page_content}
	json.Unmarshal(page_metadata, &page.Metadata)
	page.sanitize_page_name()

	if page.Name == nil {
		http.Redirect(w, r, "../view/Main", http.StatusNotFound)
		return
	}

	page.save()

	http.Redirect(w, r, "../view/" + string(page.Name), http.StatusFound)
}

func delete(w http.ResponseWriter, r *http.Request) {
	page_name := get_page_name_from_url(r, "delete")
	page := Page{Name: []byte(page_name)}
	page.delete()
	http.Redirect(w, r, "../view/Main", http.StatusFound)
}

func all_pages(w http.ResponseWriter, r *http.Request) {
	pages := get_all_pages(DATA_FOLDER)
	data := map[string]interface {} {
		"pages": pages,
	}
	render_template(w, "all_pages", data)
}

func main() {
	server_addr := "localhost:" + LOCAL_SERVER_PORT
	full_server_addr := "http://" + server_addr

	fmt.Println(DATA_FOLDER)

	fmt.Println("AlDftr v" + __VERSION__)
	fmt.Println("Please keep this window open")
	fmt.Println("-----------------------------")
	fmt.Println("AlDftr running on:" + full_server_addr)
	get_all_pages(DATA_FOLDER)

	switch runtime.GOOS {
	case "windows":
		exec.Command("cmd", "/c", "start", full_server_addr).Start()
	case "darwin":
		exec.Command("open", full_server_addr).Start()
	case "linux":
		exec.Command("xdg-open", full_server_addr).Start()
	}

	http.HandleFunc("/", index)
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir(STATIC_PATH))))
	http.HandleFunc("/view/", view)
	http.HandleFunc("/edit/", edit)
	http.HandleFunc("/save/", save)
	http.HandleFunc("/delete/", delete)
	http.HandleFunc("/all_pages/", all_pages)

	http.ListenAndServe(server_addr, nil)
}
