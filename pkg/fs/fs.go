package fs

import (
	"embed"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

type (
	Type        int
	Match       int
	PathChecker []PathCheck
)

type File[T any] struct {
	Path    T
	Content []byte
}

type FileInfo struct {
	Path string
	Info fs.FileInfo
}

type WalkDirOptions struct {
	Relative bool
	Dirs     bool
}

type PathCheck struct {
	Type   string
	Target string
	Action string
}

type EFS struct {
	Root   string
	FS     embed.FS
	Prefix string
}

const (
	MkDirAllPerm  = 0o755
	WriteFilePerm = 0o644
)

const (
	TypeFile Type = iota
	TypeDir
	TypeNone
)

const (
	MatchFile Match = iota
	MatchDirectory
	MatchAny
)

const (
	PathCheckTypeEndsWith  string = "EndsWith"
	PathCheckTypeContains  string = "Contains"
	PathCheckTypeDriveRoot string = "DriveRoot"

	PathCheckActionWarn string = "Warn"
	PathCheckActionDeny string = "Deny"
)

var ReservedHostnames = []string{
	"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
	"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	"PRN", "AUX", "NUL",
}

func FS(name string) fs.FS     { return os.DirFS(name) }
func Exists(name string) bool  { _, err := os.Stat(name); return !os.IsNotExist(err) }
func Ensure(name string) error { return os.MkdirAll(filepath.Dir(name), MkDirAllPerm) }
func Empty(name string) (bool, error) {
	file, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer file.Close()

	_, err = file.Readdir(1)
	if err == nil {
		return false, nil
	}

	if errors.Is(err, os.ErrNotExist) || err.Error() == "EOF" {
		return true, nil
	}

	return false, err
}

func Write(name string, data []byte) error { return os.WriteFile(name, data, WriteFilePerm) }
func Read(name string) ([]byte, error)     { return os.ReadFile(name) }
func Overwrite(file *os.File, size, offset int64, whence int) error {
	if err := file.Truncate(size); err != nil {
		return err
	}

	_, err := file.Seek(offset, whence)

	return err
}

func CopyFile(pathA, pathB string) (int64, error) {
	src, err := os.Open(pathA)
	if err != nil {
		return 0, err
	}
	defer src.Close()

	dst, err := os.Create(pathB)
	if err != nil {
		return 0, err
	}
	defer dst.Close()

	return io.Copy(dst, src)
}

func GetType(name string) (Type, error) {
	i, err := os.Stat(name)
	if err != nil {
		return TypeNone, err
	}

	if i.IsDir() {
		return TypeDir, nil
	}

	if i.Mode().IsRegular() {
		return TypeFile, nil
	}

	return TypeNone, nil
}

func WalkFS(e fs.FS, root string, opts *WalkDirOptions) (*[]FileInfo, error) {
	root = filepath.Clean(root)
	result := []FileInfo{}

	err := fs.WalkDir(e, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		if !opts.Dirs && d.IsDir() {
			return nil
		}

		rel := path
		if opts.Relative {
			rel, err = filepath.Rel(root, path)
			if err != nil {
				return err
			}
		}

		result = append(result, FileInfo{
			Path: rel,
			Info: info,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func WalkDir(name string, opts *WalkDirOptions) (*[]File[string], error) {
	name = filepath.Clean(name)
	result := []File[string]{}

	err := filepath.WalkDir(name, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !opts.Dirs && d.IsDir() {
			return nil
		}

		rel := path
		if opts.Relative {
			if rel, err = filepath.Rel(name, path); err != nil {
				return err
			}
		}

		data, err := Read(path)
		if err != nil {
			return err
		}

		result = append(result, File[string]{
			Path:    filepath.ToSlash(rel),
			Content: data,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func Find(name string, match Match) []string {
	result := []string{}

	err := filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		switch match {
		case MatchFile:
			if !info.IsDir() {
				result = append(result, path)
			}
		case MatchDirectory:
			if info.IsDir() {
				result = append(result, path)
			}
		case MatchAny:
			result = append(result, path)
		}

		return nil
	})
	if err != nil {
		return []string{}
	}

	return Sort(result)
}

func Rmdir(name string, skip func(string) bool) error {
	return filepath.Walk(name, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if path == name {
			return nil
		}

		if !info.IsDir() {
			if !skip(path) {
				return os.Remove(path)
			}

			return nil
		}

		return nil
	})
}

func RmdirEmpty(name string) error {
	var directories []string

	err := filepath.WalkDir(name, func(path string, directory os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if directory.IsDir() && path != name {
			directories = append(directories, path)
		}

		return nil
	})
	if err != nil {
		return err
	}

	for i := len(directories) - 1; i >= 0; i-- {
		d := directories[i]

		empty, err := Empty(d)
		if err != nil {
			continue
		}

		if empty {
			err = os.Remove(d)
			if err != nil {
				continue
			}
		}
	}

	return nil
}

func (e *EFS) List(
	name, pattern string,
	opts *WalkDirOptions,
	list func(FileInfo, []byte) error,
) error {
	files, err := WalkFS(e.FS, pattern, opts)
	if err != nil {
		return err
	}

	for _, file := range *files {
		if file.Info.IsDir() {
			continue
		}

		if !strings.Contains(file.Path, name) {
			continue
		}

		data, err := e.Read(strings.TrimPrefix(file.Path, e.Root))
		if err != nil {
			return err
		}

		if err := list(file, data); err != nil {
			return err
		}
	}

	return nil
}

func (e *EFS) ListAll(
	pattern string,
	opts *WalkDirOptions,
	list func(*[]FileInfo) error,
) error {
	files, err := WalkFS(e.FS, pattern, opts)
	if err != nil {
		return err
	}

	return list(files)
}

func (e *EFS) Bytes(name string) []byte {
	data, _ := e.Read(name)
	if data == nil {
		return []byte{}
	}

	return data
}

func (e *EFS) String(name string) string {
	data, _ := e.Read(name)
	if data == nil {
		return ""
	}

	return string(data)
}

func (e *EFS) Read(name string) ([]byte, error) {
	return e.FS.ReadFile(filepath.ToSlash(filepath.Join(e.Root, name)))
}

func Normalize(name string) string { return strings.ReplaceAll(name, "\\", "/") }
func Slice(name string) []string   { return strings.Split(Normalize(name), "/") }
func Relative(names ...string) string {
	result := "./" + names[0]
	for _, dir := range names[1:] {
		result = path.Join(result, dir)
	}

	return result
}

func Trim(name string) string {
	if strings.HasPrefix(name, "./") || strings.HasPrefix(name, ".\\") {
		return name[2:]
	} else if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "\\") {
		return name[1:]
	}

	if strings.HasSuffix(name, "/.") || strings.HasSuffix(name, "\\.") {
		return name[:len(name)-2]
	} else if strings.HasSuffix(name, "/") || strings.HasSuffix(name, "\\") {
		return name[:len(name)-1]
	}

	return name
}

func Sort(names []string) []string {
	sort.Slice(names, func(i, j int) bool {
		p1 := filepath.Dir(names[i])
		p2 := filepath.Dir(names[j])

		if p1 == p2 {
			return filepath.Base(names[i]) < filepath.Base(names[j])
		}

		return p1 < p2
	})

	return names
}

func Env(key string, values []string) string {
	separator := string(filepath.ListSeparator)
	existing := os.Getenv(key)
	joined := strings.Join(values, separator)

	if existing != "" {
		return key + "=" + joined + separator + existing
	}

	return key + "=" + joined
}

func Hostname(name string) bool {
	if len(name) < 1 || len(name) > 15 {
		return false
	}

	re := regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	if !re.MatchString(name) {
		return false
	}

	for _, reserved := range ReservedHostnames {
		if strings.EqualFold(name, reserved) {
			return false
		}
	}

	return true
}

func URLFilename(name string) (string, error) {
	u, err := url.Parse(name)
	if err != nil {
		return "", err
	}

	return path.Base(u.Path), nil
}

func FromCwd(names ...string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	path := append([]string{wd}, names...)

	return filepath.Join(path...), nil
}

func Depth(name string) int {
	if name == "." || name == "" {
		return 0
	}

	cur := 0

	for {
		dir, file := filepath.Split(name)
		if file != "" {
			cur++
		}

		if dir == "" || dir == "/" || dir == "." {
			break
		}

		name = dir[:len(dir)-1]
	}

	return cur
}

func Subpath(src, dst string) bool {
	rel, err := filepath.Rel(filepath.Clean(src), filepath.Clean(dst))
	return err == nil && !strings.HasPrefix(rel, "..")
}

func (p *PathChecker) CheckPathForProblemLocations(name string) (bool, PathCheck) {
	name = strings.ToLower(Normalize(Trim(name)))
	parts := strings.Split(name, "/")
	check := PathCheck{}

	for _, check := range *p {
		switch check.Type {
		case PathCheckTypeEndsWith:
			if strings.EqualFold(
				strings.ToLower(parts[len(parts)-1]),
				strings.ToLower(check.Target),
			) {
				return true, check
			}
		case PathCheckTypeContains:
			if slices.Contains(parts, strings.ToLower(check.Target)) {
				return true, check
			}
		case PathCheckTypeDriveRoot:
			return regexp.MustCompile(`^\w:(\\|\/)$`).Match([]byte(name)), check
		}
	}

	return false, check
}

func NewDefaultProblemPaths() []PathCheck {
	return []PathCheck{
		{Type: PathCheckTypeEndsWith, Target: "SteamApps", Action: PathCheckActionWarn},
		{Type: PathCheckTypeEndsWith, Target: "Documents", Action: PathCheckActionWarn},
		{Type: PathCheckTypeEndsWith, Target: "Desktop", Action: PathCheckActionDeny},
		{Type: PathCheckTypeContains, Target: "Desktop", Action: PathCheckActionWarn},
		{Type: PathCheckTypeContains, Target: "scoped_dir", Action: PathCheckActionDeny},
		{Type: PathCheckTypeContains, Target: "Downloads", Action: PathCheckActionDeny},
		{Type: PathCheckTypeContains, Target: "OneDrive", Action: PathCheckActionDeny},
		{Type: PathCheckTypeContains, Target: "NextCloud", Action: PathCheckActionDeny},
		{Type: PathCheckTypeContains, Target: "DropBox", Action: PathCheckActionDeny},
		{Type: PathCheckTypeContains, Target: "Google", Action: PathCheckActionDeny},
		{Type: PathCheckTypeContains, Target: "Program Files", Action: PathCheckActionDeny},
		{Type: PathCheckTypeContains, Target: "Program Files (x86)", Action: PathCheckActionDeny},
		// {Type: PathCheckTypeContains, Target: "Windows", Action: PathCheckActionDeny},
		{Type: PathCheckTypeDriveRoot, Target: "", Action: PathCheckActionDeny},

		// Reserved.
		{Type: PathCheckTypeEndsWith, Target: "CON", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "PRN", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "AUX", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "CLOCK$", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "NUL", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "COM0", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "COM1", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "COM2", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "COM3", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "COM4", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "COM5", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "COM6", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "COM7", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "COM8", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "COM9", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "LPT0", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "LPT1", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "LPT2", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "LPT3", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "LPT4", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "LPT5", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "LPT6", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "LPT7", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "LPT8", Action: PathCheckActionDeny},
		{Type: PathCheckTypeEndsWith, Target: "LPT9", Action: PathCheckActionDeny},
	}
}
