package dockerx

import "os"

func osMkdirAll(path string) error {
	return os.MkdirAll(path, 0o750)
}
