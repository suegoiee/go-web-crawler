package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type YahooNews struct {
	Title   string      `json:"title"`
	Time    string      `json:"time"`
	Content string      `json:"content"`
	Images  []NewsImage `json:"images"`
	Source  string      `json:"source"`
	Link    string      `json:"link"`
}

type NewsImage struct {
	Link        string
	Description string
}

type MongoConfig struct {
	Username string
	Password string
	Host     string
	Port     string
}

type S3Config struct {
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
}

var (
	coll     *mongo.Collection
	svc      *s3.S3
	s3Config S3Config
	imageDir string = "files/download"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("無法讀取 .env 檔案:", err)
	}

	mongoConfig := MongoConfig{
		Username: os.Getenv("USERNAME"),
		Password: os.Getenv("PASSWORD"),
		Host:     os.Getenv("HOST"),
		Port:     os.Getenv("PORT"),
	}

	credential := options.Credential{
		Username: "admin",
		Password: mongoConfig.Password,
	}

	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://" + mongoConfig.Host + ":" + mongoConfig.Port).SetAuth(credential))
	if err != nil {
		log.Fatal(err)
	}

	err = client.Connect(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	coll = client.Database("stock").Collection("news")

	s3Config = S3Config{
		Region:          os.Getenv("REGION"),
		Bucket:          os.Getenv("BUCKET"),
		AccessKeyID:     os.Getenv("S3ACCESSKEYID"),
		SecretAccessKey: os.Getenv("S3SECRETACCESSKEY"),
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(s3Config.Region),
		Credentials: credentials.NewStaticCredentials(s3Config.AccessKeyID, s3Config.SecretAccessKey, ""),
	})
	if err != nil {
		log.Fatal("Failed to create AWS session", err)
	}

	svc = s3.New(sess)
}

func main() {
	url := "https://tw.stock.yahoo.com/news/"

	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	doc1, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	results := make(map[string]bool)
	doc1.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			linkExists := strings.Contains(href, "/news/")
			if linkExists {
				domainExists := strings.Contains(href, "https://")

				if !domainExists {
					href = "https://tw.stock.yahoo.com" + href
				}
				results[href] = true
			}
		}
	})

	var wg sync.WaitGroup
	newsList := make([]interface{}, 0)
	wg.Add(len(results))

	for link := range results {
		go func(des string) {
			defer wg.Done()
			nextRes, err := http.Get(des)
			if err != nil {
				log.Fatal(err)
			}

			defer nextRes.Body.Close()
			nextDoc, err := goquery.NewDocumentFromReader(nextRes.Body)
			if err != nil {
				log.Fatal(err)
			}

			var news YahooNews
			nextDoc.Find("h1").Each(func(i int, s *goquery.Selection) {
				newsTitle, exists := s.Attr("data-test-locator")
				if newsTitle == "headline" && exists {
					news.Title = s.Text()
				}
			})
			news.Time = nextDoc.Find("time").Text()

			filter := bson.M{
				"title": news.Title,
				"time":  news.Time,
			}

			// Perform a query to check if the document already exists
			count, err := coll.CountDocuments(context.TODO(), filter)
			if err != nil {
				log.Fatal(err)
			}

			// If count is greater than zero, the document already exists
			if count > 0 {
				return
			} else {
				nextDoc.Find("noscript").Remove()
				var newsImages []NewsImage

				// 提取圖片URL並下載
				nextDoc.Find("div.caas-body img").Each(func(i int, s *goquery.Selection) {
					imgSrc, exists := s.Attr("src")
					if !exists {
						imgSrc, exists = s.Attr("data-src")
						if !exists {
							return
						}
					}

					s.AppendHtml("<p>{{img" + strconv.Itoa(i) + "}}<p>")
					// Download the image from the URL
					resp, err := http.Get(imgSrc)
					if err != nil {
						fmt.Println("Failed to download image:", err)
						return
					}
					defer resp.Body.Close()

					contentType := resp.Header.Get("Content-Type")
					fileExtension := mimeToExtension(contentType)

					// Get the filename from the URL
					filename := ""
					fileKey := ""
					if strings.LastIndex(imgSrc, "/") != -1 {
						filename = imgSrc[strings.LastIndex(imgSrc, "/")+1:] + fileExtension
						fileKey = "images/" + filename
					}

					// Create a new S3 object
					object := &s3.PutObjectInput{
						Bucket: aws.String(s3Config.Bucket),
						Key:    aws.String(fileKey),
					}
					buffer := bytes.NewBuffer(nil)

					// Copy the response body into the buffer
					_, err = io.Copy(buffer, resp.Body)
					if err != nil {
						fmt.Println("Failed to copy image data to buffer:", err)
						return
					}

					// Set the object's body to the buffer
					object.Body = bytes.NewReader(buffer.Bytes())

					// Upload the image to S3
					_, err = svc.PutObject(object)
					if err != nil {
						fmt.Println("Failed to upload image to S3:", err)
						return
					}

					publicURL := fmt.Sprintf("https://%s.s3-%s.amazonaws.com/%s", s3Config.Bucket, s3Config.Region, fileKey)
					figcaption := s.Closest("figure").Find("figcaption")
					description := figcaption.Text()
					figcaption.Remove()
					newImage := NewsImage{
						Link:        publicURL,
						Description: description,
					}
					newsImages = append(newsImages, newImage)
				})

				news.Content = nextDoc.Find("div.caas-body").Text()
				news.Images = newsImages
				news.Source = "Yahoo"
				news.Link = des
				newsList = append(newsList, news)
			}

		}(link)
	}
	wg.Wait()

	result, err := coll.InsertMany(context.TODO(), newsList)

	if err != nil {
		panic(err)
	}

	insertedCount := len(result.InsertedIDs)
	fmt.Println("Inserted count:", insertedCount)
}

func mimeToExtension(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	default:
		return ""
	}
}
