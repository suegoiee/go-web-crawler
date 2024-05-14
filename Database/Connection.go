package Database

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func MongoConnect(username string, password string, hostname string, port string, ctx context.Context) *mongo.Client {
	uri := fmt.Sprintf("mongodb://%s:%s@%s", username, password, hostname+":"+port)
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)

	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("mongodb connected!")
	}

	return client
}
