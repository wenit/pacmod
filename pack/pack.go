package pack

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Module packs the module at the given path and version then
// outputs the result to the specified output directory
func Module(path string, version string, outputDirectory string) error {
	moduleName, err := getModuleName(path)
	if err != nil {
		return fmt.Errorf("could not get module name: %w", err)
	}

	if err := createZipArchive(path, moduleName, version, outputDirectory); err != nil {
		return fmt.Errorf("could not create zip archive: %w", err)
	}

	if err := createInfoFile(version, outputDirectory); err != nil {
		return fmt.Errorf("could not create info file: %w", err)
	}

	if err := copyModuleFile(path, outputDirectory); err != nil {
		return fmt.Errorf("could not copy module file: %w", err)
	}

	return nil
}

func getModuleName(path string) (string, error) {
	moduleFilePath := filepath.Join(path, "go.mod")
	file, err := os.Open(moduleFilePath)
	defer file.Close()
	if err != nil {
		return "", fmt.Errorf("unable to open module file: %w", err)
	}

	moduleFileScanner := bufio.NewScanner(file)
	moduleFileScanner.Scan()

	moduleHeaderParts := strings.Split(moduleFileScanner.Text(), " ")
	if len(moduleHeaderParts) <= 1 {
		return "", fmt.Errorf("unable to parse module header: %w", err)
	}

	return moduleHeaderParts[1], nil
}

func createZipArchive(path string, moduleName string, version string, outputDirectory string) error {
	filePathsToArchive, err := getFilePathsToArchive(path)
	if err != nil {
		return fmt.Errorf("unable to get files to archive: %w", err)
	}

	outputPath := filepath.Join(outputDirectory, "source.zip")
	zipFile, err := os.Create(outputPath)
	defer zipFile.Close()
	if err != nil {
		return fmt.Errorf("unable to create zip file: %w", err)
	}

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, filePath := range filePathsToArchive {
		fileToZip, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("unable to open file: %w", err)
		}
		defer fileToZip.Close()

		zippedFilePath := getZipPath(path, filePath, moduleName, version)
		zippedFileWriter, err := zipWriter.Create(zippedFilePath)
		if err != nil {
			return fmt.Errorf("unable to add file to zip archive: %w", err)
		}

		if _, err := io.Copy(zippedFileWriter, fileToZip); err != nil {
			return fmt.Errorf("unable to copy file contents to zip archive: %w", err)
		}
	}

	return nil
}

func getFilePathsToArchive(path string) ([]string, error) {
	var files []string
	err := filepath.Walk(path, func(currentFilePath string, fileInfo os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("unable to walk path: %w", err)
		}

		// We do not want to include the .git directory in the archived module
		// filepath.SkipDir tells the Walk() function to ignore everything inside of the directory
		if fileInfo.IsDir() && fileInfo.Name() == ".git" {
			return filepath.SkipDir
		}

		// Do not process directories
		// returning nil tells the Walk() function to ignore this file
		if fileInfo.IsDir() {
			return nil
		}

		files = append(files, currentFilePath)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func getZipPath(path string, currentFilePath string, moduleName string, version string) string {
	filePath := strings.TrimPrefix(currentFilePath, path)
	return filepath.Join(fmt.Sprintf("%s@%s", moduleName, version), filePath)
}

func createInfoFile(version string, outputDirectory string) error {
	infoFilePath := filepath.Join(outputDirectory, version+".info")
	file, err := os.Create(infoFilePath)
	if err != nil {
		return fmt.Errorf("could not create info file: %w", err)
	}
	defer file.Close()

	type infoFile struct {
		Version string
		Time    string
	}

	currentTime := getInfoFileFormattedTime(time.Now())
	info := infoFile{
		Version: version,
		Time:    currentTime,
	}

	infoBytes, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("could not marshal info file: %w", err)
	}

	if _, err := file.Write(infoBytes); err != nil {
		return fmt.Errorf("could not write info file: %w", err)
	}

	return nil
}

func getInfoFileFormattedTime(currentTime time.Time) string {
	const infoFileTimeFormat = "2006-01-02T15:04:05Z"
	return currentTime.Format(infoFileTimeFormat)
}

func copyModuleFile(path string, outputDirectory string) error {
	if outputDirectory == "." {
		return nil
	}

	sourcePath := filepath.Join(path, "go.mod")
	destinationPath := filepath.Join(outputDirectory, "go.mod")

	if sourcePath == destinationPath {
		return nil
	}

	moduleContents, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("unable to read module file: %w", err)
	}

	if err := ioutil.WriteFile(destinationPath, moduleContents, 0644); err != nil {
		return fmt.Errorf("unable to write module file: %w", err)
	}

	return nil
}
