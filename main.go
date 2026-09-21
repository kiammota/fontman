package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	MAJOR = 1
	MINOR = 0
	PATCH = 0
)

func printVersion() {
	fmt.Printf("fontman v%d.%d.%d\n", MAJOR, MINOR, PATCH)
}

func printHelp() {
	println("fontman - font manager for Linux (Pure Go)\n")
	println("usage:")
	println("  fontman list [-u]        List system fonts (-u for user-installed only)")
	println("  fontman install <path>   Install fonts (.ttf, .otf, .zip, or directory)")
	println("  fontman grep [-u] <term> Filter installed fonts by filename/path")
	println("  fontman info <path>      Display file information for a font")
	println("  fontman help             Display this help message")
}

func isFont(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttf", ".otf", ".ttc":
		return true
	default:
		return false
	}
}

type FontManager struct {
	UserOnly bool
	fonts    []string
	loaded   bool // Indica se já buscamos as fontes no disco
}

// Retorna as fontes, buscando no disco apenas na primeira vez
func (fm *FontManager) LoadFonts() ([]string, error) {
	if fm.loaded {
		return fm.fonts, nil
	}

	var dirs []string

	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".local", "share", "fonts"))
		dirs = append(dirs, filepath.Join(home, ".fonts"))
	}

	if !fm.UserOnly {
		dirs = append(dirs, "/usr/share/fonts", "/usr/local/share/fonts")
	}

	var foundFonts []string
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && isFont(p) {
				foundFonts = append(foundFonts, p)
			}
			return nil
		})
	}

	fm.fonts = foundFonts
	fm.loaded = true
	return fm.fonts, nil
}

// Método de busca atrelado ao FontManager
func (fm *FontManager) Grep(pattern string) (string, error) {
	if strings.TrimSpace(pattern) == "" {
		return "", fmt.Errorf("search pattern cannot be empty")
	}

	fonts, err := fm.LoadFonts()
	if err != nil {
		return "", fmt.Errorf("failed to list fonts: %w", err)
	}

	var results []string
	patternLower := strings.ToLower(pattern)

	for _, font := range fonts {
		if strings.Contains(strings.ToLower(font), patternLower) {
			results = append(results, font)
		}
	}

	if len(results) == 0 {
		return "", fmt.Errorf("no results found for: %s", pattern)
	}

	return strings.Join(results, "\n"), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Close()
}

func extractZip(path, dst string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, file := range r.File {
		target := filepath.Join(dst, filepath.Clean(file.Name))

		rel, err := filepath.Rel(dst, target)
		if err != nil || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("invalid zip path: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return err
		}

		out, err := os.Create(target)
		if err != nil {
			src.Close()
			return err
		}

		_, copyErr := io.Copy(out, src)
		src.Close()
		out.Close()

		if copyErr != nil {
			return copyErr
		}
	}

	return nil
}

func Install(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	var fonts []string

	if info.IsDir() {
		err = filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && isFont(p) {
				fonts = append(fonts, p)
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else if strings.EqualFold(filepath.Ext(path), ".zip") {
		tmp, err := os.MkdirTemp("", "fontman-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)

		err = extractZip(path, tmp)
		if err != nil {
			return err
		}

		err = filepath.Walk(tmp, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && isFont(p) {
				fonts = append(fonts, p)
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else if isFont(path) {
		fonts = append(fonts, path)
	} else {
		return fmt.Errorf("unsupported file: %s", path)
	}

	if len(fonts) == 0 {
		return fmt.Errorf("no fonts found")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	fontDir := filepath.Join(home, ".local", "share", "fonts")
	if err := os.MkdirAll(fontDir, 0755); err != nil {
		return err
	}

	for _, font := range fonts {
		dst := filepath.Join(fontDir, filepath.Base(font))
		if err := copyFile(font, dst); err != nil {
			return err
		}
		fmt.Printf("installed: %s\n", filepath.Base(font))
	}

	return nil
}

func main() {
	if runtime.GOOS != "linux" {
		fmt.Fprintln(os.Stderr, "fontman: unsupported operating system")
		os.Exit(1)
	}

	if len(os.Args) < 2 {
		printHelp()
		return
	}

	cmd := os.Args[1]

	if cmd == "-v" || cmd == "--version" {
		printVersion()
		return
	}
	if cmd == "-h" || cmd == "--help" {
		printHelp()
		return
	}

	userOnly := false
	var commandArgs []string

	for _, arg := range os.Args[2:] {
		if arg == "-u" || arg == "--user" {
			userOnly = true
		} else {
			commandArgs = append(commandArgs, arg)
		}
	}

	manager := &FontManager{
		UserOnly: userOnly,
	}

	switch cmd {
	case "list":
		fonts, err := manager.LoadFonts()
		if err != nil {
			println("failed to scan fonts directories")
			return
		}

		for _, font := range fonts {
			fmt.Println(font)
		}

	case "install":
		if len(commandArgs) == 0 {
			println("require <path> argument")
			return
		}

		if err := Install(commandArgs[0]); err != nil {
			println("failed to install fonts:", err.Error())
			return
		}
		println("fonts installed successfully")

	case "grep":
		if len(commandArgs) == 0 {
			println("require <pattern> argument")
			return
		}

		pattern := commandArgs[0]
		s, err := manager.Grep(pattern)
		if err != nil {
			println(err.Error())
			return
		}
		println(s)

	case "info":
		if len(commandArgs) == 0 {
			println("require <font_path_or_name> argument")
			return
		}

		searchTerm := commandArgs[0]
		fontPath := searchTerm

		// Tenta estatísticas diretas do arquivo primeiro
		info, err := os.Stat(fontPath)

		if err != nil {
			// Usa o cache interno via LoadFonts
			fonts, errList := manager.LoadFonts()
			if errList == nil {
				searchTermNormalized := strings.ReplaceAll(strings.ToLower(searchTerm), " ", "")

				for _, f := range fonts {
					fileNameNormalized := strings.ReplaceAll(strings.ToLower(filepath.Base(f)), " ", "")
					if strings.Contains(fileNameNormalized, searchTermNormalized) {
						fontPath = f
						info, err = os.Stat(fontPath)
						break
					}
				}
			}
		}

		if err != nil {
			fmt.Printf("font not found: %s\n", searchTerm)
			return
		}

		fmt.Printf("File: %s\n", fontPath)
		fmt.Printf("Size: %d bytes\n", info.Size())
		fmt.Printf("Mode: %s\n", info.Mode().String())
		fmt.Printf("Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		fmt.Println("\n(Note: Extracting internal font family/style data requires a TTF/OTF binary parser instead of fontconfig)")
	
	default:
		fmt.Printf("unknown command: %s\n", cmd)
		printHelp()
	}
}
