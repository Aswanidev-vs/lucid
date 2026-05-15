package handler

import (
	"html/template"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	tmplCache = make(map[string]*template.Template)
	tmplMu    sync.RWMutex
)

var FuncMap = template.FuncMap{
	"formatTime": func(t interface{}) string {
		switch v := t.(type) {
		case time.Time:
			return v.Format("Jan 2, 2006 3:04 PM")
		case string:
			t, err := time.Parse("2006-01-02T15:04:05Z", v)
			if err != nil {
				t, err = time.Parse("2006-01-02 15:04:05", v)
				if err != nil {
					return v
				}
			}
			return t.Format("Jan 2, 2006")
		default:
			return ""
		}
	},
	"firstChar": func(s string) string {
		if len(s) == 0 {
			return "?"
		}
		return string([]rune(s)[:1])
	},
	"formatTimestamp": func(t time.Time) string {
		return t.Format("Jan 2, 2006 3:04 PM")
	},
}

func ParseTemplate(files ...string) (*template.Template, error) {
	if len(files) == 0 {
		return nil, nil
	}

	name := files[0]
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '/' || name[i] == '\\' {
			name = name[i+1:]
			break
		}
	}

	tmplMu.RLock()
	if cached, ok := tmplCache[name]; ok {
		tmplMu.RUnlock()
		return cached, nil
	}
	tmplMu.RUnlock()

	tmplMu.Lock()
	defer tmplMu.Unlock()

	if cached, ok := tmplCache[name]; ok {
		return cached, nil
	}

	// Resolve paths: if the file doesn't include a directory separator,
	// assume it's in the templates/ directory, EXCEPT for index.html
	// which lives in the project root.
	paths := make([]string, len(files))
	for i, f := range files {
		if strings.Contains(f, "/") || strings.Contains(f, "\\") {
			paths[i] = f
		} else if f == "index.html" {
			paths[i] = f
		} else {
			paths[i] = filepath.Join("templates", f)
		}
	}

	t, err := template.New(name).Funcs(FuncMap).ParseFiles(paths...)
	if err != nil {
		return nil, err
	}

	tmplCache[name] = t
	return t, nil
}
