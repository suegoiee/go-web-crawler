package AWSStorage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

var (
	s3Config S3Config
	err      error
)

func SaveImage(resp *http.Response, imageURL string, imageName string, svc *s3.S3) string {

	// Create a new S3 object
	object := &s3.PutObjectInput{
		Bucket: aws.String(s3Config.Bucket),
		Key:    aws.String(imageName),
	}
	buffer := bytes.NewBuffer(nil)

	// Copy the response body into the buffer
	_, err = io.Copy(buffer, resp.Body)
	if err != nil {
		fmt.Println("Failed to copy image data to buffer:", err)
		return ""
	}

	// Set the object's body to the buffer
	object.Body = bytes.NewReader(buffer.Bytes())

	// Upload the image to S3
	_, err = svc.PutObject(object)
	if err != nil {
		fmt.Println("Failed to upload image to S3:", err)
		return ""
	}

	return fmt.Sprintf("https://%s.s3-%s.amazonaws.com/%s", s3Config.Bucket, s3Config.Region, "images/"+imageName)
}
