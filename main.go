package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"sync"
	"time"

	"josh/goCrawler/AWSStorage"
	"josh/goCrawler/Database"
	"josh/goCrawler/LocalStorage"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/joho/godotenv"

	"go.mongodb.org/mongo-driver/mongo"
)

type News struct {
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
type Results struct {
	news    News
	urlList []string
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
	coll      *mongo.Collection
	svc       *s3.S3
	s3Config  S3Config
	mode      string
	publicURL string
)

func main() {
	err := godotenv.Load()
	firstAttempt := Results{}
	if err != nil {
		log.Fatal("Unable to load .env:", err)
	}

	username := os.Getenv("MONGOUSERNAME")
	password := os.Getenv("MONGOPASSWORD")
	hostname := os.Getenv("MONGOHOST")
	port := os.Getenv("MONGOPORT")
	mongoDatabase := os.Getenv("MONGODATABASE")
	mongoCollection := os.Getenv("MONGOCOLLECTION")
	mode = os.Getenv("IMAGEMODE")
	imageDir := os.Getenv("LOCALIMAGEPATH")
	loop, err := strconv.Atoi(os.Getenv("SCRAPELOOP"))
	if err != nil {
		// ... handle error
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()
	client := Database.MongoConnect(username, password, hostname, port, ctx)
	coll = client.Database(mongoDatabase).Collection(mongoCollection)

	s3Config = S3Config{
		Region:          os.Getenv("REGION"),
		Bucket:          os.Getenv("BUCKET"),
		AccessKeyID:     os.Getenv("S3ACCESSKEYID"),
		SecretAccessKey: os.Getenv("S3SECRETACCESSKEY"),
	}

	if mode == "aws" {
		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String(s3Config.Region),
			Credentials: credentials.NewStaticCredentials(s3Config.AccessKeyID, s3Config.SecretAccessKey, ""),
		})
		if err != nil {
			log.Fatal("Failed to create AWS session", err)
		} else {
			fmt.Println("AWS session created!")
		}

		svc = s3.New(sess)
	}

	defer cancel()
	workerPoolSize := 10
	workerPool := make(chan struct{}, workerPoolSize)
	var wg sync.WaitGroup
	wg.Add(1)

	url := os.Getenv("CRAWLURL")
	firstAttempt = getNews(url, imageDir, workerPool, &wg, true)

	newsCh := make(chan News)
	for i := 0; i < loop; i++ {
		tmp := Results{}
		newsList := make([]interface{}, 0)
		todo := firstAttempt.urlList
		for _, link := range todo {
			wg.Add(1)
			go func(newsLink string) {
				tmp = getNews(newsLink, imageDir, workerPool, &wg, true)
				todo = append(todo, tmp.urlList...)
				newsCh <- tmp.news
			}(link)
			newsList = append(newsList, <-newsCh)
		}

		result, err := coll.InsertMany(context.TODO(), newsList)

		if err != nil {
			panic(err)
		} else {
			fmt.Println("Data successfully inserted")
		}

		insertedCount := len(result.InsertedIDs)
		fmt.Println("Inserted count:", insertedCount)
	}
	go func() {
		close(newsCh)
		wg.Wait()
	}()
}

func getNews(url string, imgDir string, workerPool chan struct{}, wg *sync.WaitGroup, gatherUrlFlag bool) Results {
	defer wg.Done()
	workerPool <- struct{}{}
	result := Results{}
	urlList := []string{}
	res, err := http.Get(url)

	if err != nil {
		fmt.Println("http.Get error!")
		log.Fatal(err)
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		fmt.Println("NewDocumentFromReader error!")
		log.Fatal(err)
	}

	if gatherUrlFlag {
		urlList = RetrieveUrlList(doc, url)
	}

	var news News
	doc.Find("h1").Each(func(i int, s *goquery.Selection) {
		newsTitle, exists := s.Attr("data-test-locator")
		if newsTitle == "headline" && exists {
			news.Title = s.Text()
		}
	})
	news.Time = doc.Find("time").Text()
	doc.Find("noscript").Remove()
	var newsImages []NewsImage

	// 提取圖片URL並下載
	doc.Find("div.caas-body img").Each(func(i int, s *goquery.Selection) {
		imgSrc, exists := s.Attr("src")
		if !exists {
			imgSrc, exists = s.Attr("data-src")
			if !exists {
				fmt.Println("image not found")
				<-workerPool
				return
			}
		}

		s.AppendHtml("<p>{{img" + strconv.Itoa(i) + "}}<p>")
		// Download the image from the URL
		resp, err := http.Get(imgSrc)
		if err != nil {
			fmt.Println("Failed to download image:", err)
			<-workerPool
			return
		}
		defer resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")
		fileExtension := mimeToExtension(contentType)

		// Get the filename from the URL
		filename := ""
		if strings.LastIndex(imgSrc, "/") != -1 {
			filename = imgSrc[strings.LastIndex(imgSrc, "/")+1:] + fileExtension
		}

		if mode == "aws" {
			publicURL = AWSStorage.SaveImage(resp, imgSrc, filename, svc)
		} else if mode == "local" {
			publicURL = LocalStorage.SaveImage(imgSrc, filename, imgDir)
		} else {
			panic("Image download mode not set.")
		}

		figcaption := s.Closest("figure").Find("figcaption")
		description := figcaption.Text()
		figcaption.Remove()
		newImage := NewsImage{
			Link:        publicURL,
			Description: description,
		}
		newsImages = append(newsImages, newImage)
	})

	news.Content = doc.Find("div.caas-body").Text()
	news.Images = newsImages
	news.Source = os.Getenv("CRAWLURLSOURCE")
	news.Link = url

	result.news = news
	result.urlList = urlList
	<-workerPool
	return result
}

func RetrieveUrlList(doc *goquery.Document, domain string) []string {
	result := []string{}
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && linkFilter(href) {
			if !strings.Contains(href, "https") {
				href = domain + href
			}
			result = append(result, href)
		}
	})

	return result
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

// Filter for ads and relative path
func linkFilter(href string) bool {
	// Check if the link is a news link based on specific criteria
	linkExists := strings.Contains(href, "/news/")
	domainExists := strings.Contains(href, "https://")

	// Filter out unnecessary content by defining your criteria
	if linkExists && domainExists {
		return true
	} else if !domainExists && linkExists {
		return true
	}

	return false
}
