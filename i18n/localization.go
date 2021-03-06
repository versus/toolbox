// Package i18n provides internationalization support for applications.
package i18n

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/richardwilkes/toolbox/log/logadapter"
	"github.com/richardwilkes/toolbox/xio/fs"
)

const (
	// Extension is the file name extension required on localization files.
	Extension = ".i18n"
)

var (
	// Dir is the directory to scan for localization files. This will occur
	// only once, the first time a call to Text() is made. If you do not set
	// this prior to the first call, a directory in the same location as the
	// executable with "_i18n" appended to the executable name (sans any
	// extension) will be used.
	Dir string
	// Language is the language that should be used for text returned from
	// calls to Text(). It is initialized to the result of calling Locale().
	// You may set this at runtime, forcing a particular language for all
	// subsequent calls to Text().
	Language = Locale()
	// Languages is a slice of languages to fall back to should the one
	// specified in the Language variable not be available. If is initialized
	// to the value of the LANGUAGE environment variable.
	Languages = strings.Split(os.Getenv("LANGUAGE"), ":")
	// Log is set to discard by default.
	Log      logadapter.ErrorLogger = &logadapter.Discarder{}
	once     sync.Once
	langMap  = make(map[string]map[string]string)
	hierLock sync.Mutex
	hierMap  = make(map[string][]string)
)

// Text returns a localized version of the text if one exists, or the original
// text if not.
func Text(text string) string {
	once.Do(func() {
		if Dir == "" {
			path, err := os.Executable()
			if err != nil {
				return
			}
			path, err = filepath.EvalSymlinks(path)
			if err != nil {
				return
			}
			path, err = filepath.Abs(fs.TrimExtension(path) + "_i18n")
			if err != nil {
				return
			}
			Dir = path
		}
		fi, err := ioutil.ReadDir(Dir)
		if err != nil {
			return
		}
		for _, one := range fi {
			if !one.IsDir() {
				name := one.Name()
				if filepath.Ext(name) == Extension {
					load(name)
				}
			}
		}
	})

	var result string
	if result = lookup(text, Language); result != "" {
		return result
	}
	for _, language := range Languages {
		if result = lookup(text, language); result != "" {
			return result
		}
	}
	return text
}

func lookup(text, language string) string {
	for _, lang := range hierarchy(language) {
		if translations := langMap[lang]; translations != nil {
			if str, ok := translations[text]; ok {
				return str
			}
		}
	}
	return ""
}

func hierarchy(language string) []string {
	lang := strings.ToLower(language)
	hierLock.Lock()
	defer hierLock.Unlock()
	if s, ok := hierMap[lang]; ok {
		return s
	}
	one := lang
	var s []string
	for {
		s = append(s, one)
		if i := strings.LastIndexAny(one, "._"); i != -1 {
			one = one[:i]
		} else {
			break
		}
	}
	hierMap[lang] = s
	return s
}

func load(name string) {
	path := filepath.Join(Dir, name)
	if file, err := os.Open(path); err == nil {
		translations := make(map[string]string)
		langMap[strings.ToLower(name[:len(name)-len(Extension)])] = translations
		var key string
		var lineNum int
		var lastKeyLineNum int
		in := bufio.NewReader(file)
		for {
			var line string
			line, err = in.ReadString('\n')
			lineNum++
			if strings.HasPrefix(line, "k:") {
				if key != "" {
					Log.Errorf("value missing for key on line %d of %s\n", lastKeyLineNum, path)
				}
				if key = extract(line, "key", lineNum, path); key != "" {
					lastKeyLineNum = lineNum
				}
			} else if strings.HasPrefix(line, "v:") {
				if key == "" {
					Log.Errorf("no key for value on line %d of %s\n", lineNum, path)
				} else if value := extract(line, "value", lineNum, path); value != "" {
					translations[key] = value
					key = ""
				}
			}
			if err != nil {
				break
			}
		}
		if err = file.Close(); err != nil {
			Log.Error(err)
		}
	}
}

func extract(line, label string, lineNum int, path string) string {
	str, err := strconv.Unquote(strings.TrimSpace(line[2:]))
	if err != nil {
		Log.Errorf("%s malformed on line %d of %s\n", label, lineNum, path)
		return ""
	}
	return str
}
