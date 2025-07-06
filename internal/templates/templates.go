package templates

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func LoadTemplates() (*template.Template, error) {
	tmpl := template.New("").Funcs(template.FuncMap{})

	err := filepath.Walk("views", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".html") {
			bytes, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			name := strings.TrimPrefix(
				strings.ReplaceAll(path, "\\", "/"),
				"views/",
			)

			if _, err := tmpl.New(name).Parse(string(bytes)); err != nil {
				log.Fatalf("failed to parse %s error:%v", name, err)
			}
		}
		return nil
	})

	return tmpl, err
}
