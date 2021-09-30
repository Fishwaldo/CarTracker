package update

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/bzip2"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"errors"

//	"github.com/pkg/errors"
)

func findHash(buf []byte, filename string) (hash []byte, err error) {
	sc := bufio.NewScanner(bytes.NewReader(buf))
	for sc.Scan() {
		data := strings.Split(sc.Text(), "  ")
		if len(data) != 2 {
			continue
		}

		if data[1] == filename {
			h, err := hex.DecodeString(data[0])
			if err != nil {
				return nil, err
			}

			return h, nil
		}
	}

	return nil, fmt.Errorf("hash for file %v not found", filename)
}

func extractToFile(buf []byte, filename, target string) error {
	var mode = os.FileMode(0755)

	// get information about the target file
	fi, err := os.Lstat(target)
	if err == nil {
		mode = fi.Mode()
	}

	var rd io.Reader = bytes.NewReader(buf)
	switch filepath.Ext(filename) {
	case ".bz2":
		rd = bzip2.NewReader(rd)
	case ".zip":
		zrd, err := zip.NewReader(bytes.NewReader(buf), int64(len(buf)))
		if err != nil {
			return err
		}

		if len(zrd.File) != 1 {
			return errors.New("ZIP archive contains more than one file")
		}

		file, err := zrd.File[0].Open()
		if err != nil {
			return err
		}

		defer func() {
			_ = file.Close()
		}()

		rd = file
	}

	err = os.Remove(target)
	if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		return fmt.Errorf("unable to remove target file: %v", err)
	}

	dest, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}

	n, err := io.Copy(dest, rd)
	if err != nil {
		_ = dest.Close()
		_ = os.Remove(dest.Name())
		return err
	}

	err = dest.Close()
	if err != nil {
		return err
	}

	fmt.Printf("saved %d bytes in %v\n", n, dest.Name())
	return nil
}

// DownloadLatestStableRelease downloads the latest stable released version of
// restic and saves it to target. It returns the version string for the newest
// version. The function printf is used to print progress information.
func DownloadLatestStableRelease(ctx context.Context, target, currentVersion string) (version string, err error) {

	fmt.Printf("find latest release of restic at GitHub\n")

	rel, err := GitHubLatestRelease(ctx, "Fishwaldo", "CarTracker")
	if err != nil {
		return "", err
	}

	if rel.Version == currentVersion {
		fmt.Printf("restic is up to date\n")
		return currentVersion, nil
	}

	fmt.Printf("latest version is %v\n", rel.Version)

	_, sha256sums, err := getGithubDataFile(ctx, rel.Assets, "SHA256SUMS")
	if err != nil {
		return "", err
	}

	// _, sig, err := getGithubDataFile(ctx, rel.Assets, "SHA256SUMS.asc", printf)
	// if err != nil {
	// 	return "", err
	// }

	// // ok, err := GPGVerify(sha256sums, sig)
	// if err != nil {
	// 	return "", err
	// }

	// if !ok {
	// 	return "", errors.New("GPG signature verification of the file SHA256SUMS failed")
	// }

	// printf("GPG signature verification succeeded\n")

	ext := "bz2"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}

	suffix := fmt.Sprintf("%s_%s.%s", runtime.GOOS, runtime.GOARCH, ext)
	downloadFilename, buf, err := getGithubDataFile(ctx, rel.Assets, suffix)
	if err != nil {
		return "", err
	}

	fmt.Printf("downloaded %v\n", downloadFilename)

	wantHash, err := findHash(sha256sums, downloadFilename)
	if err != nil {
		return "", err
	}

	gotHash := sha256.Sum256(buf)
	if !bytes.Equal(wantHash, gotHash[:]) {
		return "", fmt.Errorf("SHA256 hash mismatch, want hash %02x, got %02x", wantHash, gotHash)
	}

	err = extractToFile(buf, downloadFilename, target)
	if err != nil {
		return "", err
	}

	return rel.Version, nil
}

func DoUpdate(version string) (err error) {
	file, err := os.Executable()
	if err != nil {
		return errors.New("unable to find executable")
	}
	fi, err := os.Lstat(file)
	if err != nil {
		dirname := filepath.Dir(file)
		di, err := os.Lstat(dirname)
		if err != nil {
			return err
		}
		if !di.Mode().IsDir() {
			return fmt.Errorf("output parent path %v is not a directory, use --output to specify a different file path", dirname)
		}
	} else {
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("output path %v is not a normal file, use --output to specify a different file path", file)
		}
	}

	fmt.Printf("writing CarTracker to %v\n", file)

	v, err := DownloadLatestStableRelease(context.Background(), file, version)
	if err != nil {
		return fmt.Errorf("unable to update restic: %v", err)
	}

	if v != version {
		fmt.Printf("successfully updated restic to version %v\n", v)
	}
	return nil
}