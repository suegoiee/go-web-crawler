# go-web-crawler (news)

Welcome to the Web Crawler Project! This repository showcases a news website crawler built using Go. The project focuses on web scraping, data storage, concurrency with Goroutines, and integration with MongoDB and AWS S3.

## Table of Contents

- [Introduction](#introduction)
- [Features](#features)
- [Technologies Used](#technologies-used)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)

## Introduction

The Web Crawler Project demonstrates a news website crawler written in Go. This project aims to extract news articles from various websites efficiently. It utilizes the GoQuery library for parsing HTML, MongoDB for storing article data, and AWS S3 for hosting article images. The project employs Goroutines for concurrent processing, enhancing crawling speed.

## Features

- Efficiently crawls news articles from websites.
- Utilizes GoQuery for parsing HTML content.
- Stores extracted article data in MongoDB.
- Implements concurrent crawling with Goroutines.
- Hosts article images on AWS S3.
- Secures sensitive data using `.env` configuration.

## Technologies Used

- Go: Core programming language.
- GoQuery: Library for HTML parsing.
- MongoDB: Database for article storage.
- AWS S3: Cloud storage for images.
- Goroutines: Concurrency for efficient crawling.
- `.env` file: Secure configuration storage.

## Getting Started

1. Clone the repository:

```bash
git clone https://github.com/suegoiee/news-crawler.git
cd go-web-crawler
```
   
2. Install required dependencies:

  ```bash
  go get -u github.com/PuerkitoBio/goquery
  go get -u go.mongodb.org/mongo-driver/mongo
  go get -u github.com/aws/aws-sdk-go
  ```
3. Configure your .env file with required settings. See the Configuration section for guidance.

Build and run the project:

  ```bash
  go build
  ./news-crawler
  ```
## Configuration

Copy the .env.example file to .env and replace values with your own:

```makefile
HOST=your-mongodb-uri
PORT=your-mongodb-port
USERNAME=your-mongodb-username
PASSWORD=your-mongodb-password

REGION=your-aws-region
S3ACCESSKEYID=your-aws-access-key-id
S3SECRETACCESSKEY=your-aws-secret-access-key
BUCKET=your-aws-s3-bucket

CRAWLURL=target-url
CRAWLURLSOURCE=target-source-name
```

## Usage

Each website's news content and image have different HTML tag, make sure to fit the correct tag and structure.

## Contributing

Feel free to fork the repository and submit pull requests to contribute.#License
This project is licensed under the MIT License.
