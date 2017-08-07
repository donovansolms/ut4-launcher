package updater

// These functions have been taken from
// https://www.socketloop.com/tutorials/golang-copy-directory-including-sub-directories-files
// with slight modifications because I am too lazy to build my own
import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies afile from source to destination and preserves permissions
func CopyFile(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourcefile.Close()

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			os.Chmod(dest, sourceinfo.Mode())
		}
	}
	return
}

// CopyDir copies a directory and all contents while preserving permissions
func CopyDir(source string, dest string) (err error) {

	// get properties of source dir
	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	// create dest dir
	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}
	directory, _ := os.Open(source)

	objects, err := directory.Readdir(-1)
	for _, obj := range objects {
		sourcefilepointer := filepath.Join(source, obj.Name())
		destinationfilepointer := filepath.Join(dest, obj.Name())
		if obj.IsDir() {
			// create sub-directories - recursively
			err = CopyDir(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			// perform copy
			err = CopyFile(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		}

	}
	return
}
