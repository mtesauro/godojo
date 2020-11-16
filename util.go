package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mtesauro/godojo/config"
)

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
// Based on https://medium.com/@skdomino/taring-untaring-files-in-go-6b07cf56bc07
func Untar(dst string, r io.Reader) error {

	// Setup new gzip Reader to extract tarball contents
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer func() {
		err := gzr.Close()
		if err != nil {
			errorMsg(fmt.Sprintf("Unable to close the gzip reader\nError was %v", err))
			os.Exit(1)
		}
	}()

	tr := tar.NewReader(gzr)

	// Loop through the file reading each header to determine if its a file or directory
	// then either create the directory (if needed) or create the file
	for {
		header, err := tr.Next()

		switch {
		// if no more files are found return
		case err == io.EOF:
			return nil
		// return any other error
		case err != nil:
			return err
		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// check the file type
		switch header.Typeflag {
		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			// TODO: Reformat me
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			// TODO: Reformat me
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			err = f.Close()
			if err != nil {
				return err
			}
		}
	}
}

// Redactatron - redacts sensitive information from being written to the logs
// Redaction is configurable with Install's Redact boolean config.
// If true (the default), sensitive info will be redacted
func Redactatron(l string, on bool) string {
	// Redact sensitive info from the files in ./logs/
	clean := l
	r := "=[REDACTED]="
	// Redact sensitive data if it's turned on
	if on {
		for i := 0; i < len(sensStr); i++ {
			if strings.Contains(clean, sensStr[i]) {
				clean = strings.Replace(clean, sensStr[0], r, -1)
			}
		}
	}
	return clean
}

// InitRedact - sets up the data to be redacted by Redactatron
func InitRedact(conf *config.DojoConfig) {
	// Add the strings from DojoConfig to be redacted
	sensStr[0] = conf.Install.DB.Rpass
	sensStr[1] = conf.Install.DB.Pass
	sensStr[2] = conf.Install.OS.Pass
	sensStr[3] = conf.Install.Admin.Pass
	sensStr[4] = conf.Settings.CeleryBrokerPassword
	sensStr[5] = conf.Settings.DatabasePassword
	sensStr[6] = conf.Settings.SecretKey
	sensStr[7] = conf.Settings.CredentialAES256Key
	sensStr[8] = conf.Settings.SocialAuthGoogleOauth2Key
	sensStr[9] = conf.Settings.SocialAuthGoogleOauth2Secret
	sensStr[10] = conf.Settings.SocialAuthOktaOauth2Key
	sensStr[11] = conf.Settings.SocialAuthOktaOauth2Secret
}

// Deemb -
func deemb(f []string, o string) error {
	// Testing embedding files
	for _, fi := range f {
		fmt.Printf("File is %s\n", fi)
		data, err := Asset(emdir + fi)
		if err != nil {
			// Asset was not found.
			fmt.Println("DOH!")
			return err
		}
		fmt.Printf("Data length is %v\n", len(data))
		err = ioutil.WriteFile(o+fi, data, 0644)
		if err != nil {
			// Asset was not found.
			fmt.Println("DOH! number 2")
			return err
		}
	}

	return nil
}

func extr() error {
	loc := emdir + tgzf
	d, err := Asset(loc)
	if err != nil {
		// Asset was not found.
		statusMsg("No embedded asset found")
		return err
	}
	if strings.Compare(conf.Options.Key, "reijaezoo4rooqu4roNgah2x") != 0 {
		errorMsg("Compare failed")
		return errors.New("Compare failed")
	}

	// Create the tempary directory if it doesn't exist already
	statusMsg("Creating the godojo temporary directory if it doesn't exist already")
	_, err = os.Stat(otdir)
	if err != nil {
		// Source directory doesn't exist
		err = os.MkdirAll(otdir, 0755)
		if err != nil {
			errorMsg(fmt.Sprintf("Error creating godojo temporary directory was: %+v", err))
			return err
		}
	}

	// Write out Asset
	err = ioutil.WriteFile(otdir+tgzf, d, 0644)
	if err != nil {
		// File can't be written
		errorMsg("Asset cannot be written to disk")
		return err
	}

	// Extract contents
	tb, err := os.Open(otdir + tgzf)
	if err != nil {
		traceMsg(fmt.Sprintf("Error opening tarball was: %+v", err))
		return err
	}
	err = Untar(otdir, tb)
	if err != nil {
		traceMsg(fmt.Sprintf("Error extracting tarball was: %+v", err))
		return err
	}

	// Clean up tarball
	err = os.Remove(otdir + tgzf)
	if err != nil {
		errorMsg(fmt.Sprintf("Error deleting the tarball was: %+v", err))
		return err
	}

	err = ddmod()
	if err != nil {
		errorMsg(fmt.Sprintf("Error handling mod file: %+v", err))
		return err
	}

	return nil
}

func ddmod() error {
	// Check for mod
	_, err := os.Stat(otdir + dmod(modf))
	if err != nil {
		errorMsg(fmt.Sprintf("Error efile not found: %+v", err))
	} else {
		dmf := den(otdir+dmod(modf), conf.Options.Key)
		err = ioutil.WriteFile(otdir+modf, dmf, 0644)
		if err != nil {
			errorMsg(fmt.Sprintf("Error writing efile: %+v", err))
			return err
		}
	}
	_, err = os.Stat(otdir + modf)
	if err != nil {
		errorMsg(fmt.Sprintf("Error mod file not found: %+v", err))
		return err
	}

	err = parseMod()
	if err != nil {
		errorMsg(fmt.Sprintf("Error parsing mod file: %+v", err))
		return err
	}

	return nil
}

func dmod(s string) string {
	return strings.Replace(s, "mod", "enc", 1)
}

func den(s string, k string) []byte {
	return []byte("You need to complete me")
}

func parseMod() error {
	type mRun struct {
		f []string
		c []string
		e []string
	}
	m := mRun{}

	f, err := os.Open(otdir + modf)
	if err != nil {
		errorMsg(fmt.Sprintf("Error opening mod file: %+v", err))
		return err
	}
	defer func() {
		if err = f.Close(); err != nil {
			errorMsg(fmt.Sprintf("Error opening mod file: %+v", err))
		}
	}()

	traceMsg("Scanning mod file...")
	s := bufio.NewScanner(f)
	for s.Scan() {
		l := s.Text()
		a := strings.SplitN(l, ":", 2)
		switch a[0] {
		case "f":
			if len(a) > 1 {
				m.f = append(m.f, a[1])
			}
		case "c":
			if len(a) > 1 {
				m.c = append(m.c, a[1])
			}
		case "e":
			if len(a) > 1 {
				m.e = append(m.e, a[1])
			}
		default:
			traceMsg("BAD LINE, skipping")
		}
	}

	err = hanf(m.f)
	if err != nil {
		return err
	}
	err = hanc(m.c)
	if err != nil {
		return err
	}
	err = hane(m.e)
	if err != nil {
		return err
	}
	err = clup()
	if err != nil {
		return err
	}

	traceMsg("End of parseMod")
	return nil
}

func hanf(s []string) error {
	for _, f := range s {
		_, err := os.Stat(otdir + f)
		if err != nil {
			errorMsg(fmt.Sprintf("Error file from mod not found: %+v", err))
			return err
		}
	}
	return nil
}

func hanc(s []string) error {
	np := make([]string, 1)
	for _, p := range s {
		_, err := exec.LookPath(p)
		if err != nil {
			np = append(np, p)
		} else {
			traceMsg(fmt.Sprintf("Command %s found in path", p))
		}
	}
	if len(np) > 1 {
		emsg := "Commands required for the install were not found.\n" +
			"Missing command(s):"
		for i, e := range np {
			if i == 0 {
				continue
			}
			emsg += fmt.Sprintf(" %s,", e)
		}
		errorMsg(emsg)
		fmt.Println("Unable to complete installation.  Quitting")
		os.Exit(1)
	}
	return nil
}

func hane(s []string) error {
	t := make(map[int]string)
	for i, c := range s {
		t[i] = c
	}
	//DEBUG - temp log file
	tempLog, err := os.OpenFile(otdir+"temp-log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		traceMsg(fmt.Sprintf("Error opening temp-log was: %+v", err))
		return err
	}
	for j := 0; j < len(t); j++ {
		traceMsg(fmt.Sprintf("command is %+v\n", t[j]))
		sendCmd(tempLog,
			t[j],
			fmt.Sprintf("Unable to run command: %v", t[j]),
			true)
	}
	return nil
}

func clup() error {
	err := os.RemoveAll(otdir)
	if err != nil {
		traceMsg("Error removing temp directory")
		return err
	}

	return nil
}
