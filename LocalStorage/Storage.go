package LocalStorage

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func SaveImage(imageURL string, imageName string, localPath string) string {
	fullPath := filepath.Join(localPath, imageName)
	absolutePath, err := filepath.Abs(fullPath)

	if err != nil {
		println("Error getting absolute path:", err.Error())
		return ""
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(absolutePath), os.ModePerm); err != nil {
		println("Error creating directory:", err.Error())
		return ""
	}

	// Create an HTTP client
	client := &http.Client{}

	// Send a GET request to the image URL
	resp, err := client.Get(imageURL)
	if err != nil {
		println(err)
		return ""
	}
	defer resp.Body.Close()

	// Check if the HTTP request was successful
	if resp.StatusCode != http.StatusOK {
		println("Failed to download image: HTTP status " + resp.Status)
		return ""
	}

	// Create a file to save the downloaded image
	file, err := os.Create(absolutePath)
	if err != nil {
		fmt.Println("Create image error.", err.Error())
		return ""
	}
	defer file.Close()

	// Copy the body of the response to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		println(err)
		return ""
	}

	println("Image successfully saved to", fullPath)
	return fullPath
}
