package Models

type News struct {
	Title   string `json:"title" bson:"title"`
	Time    string `json:"time" bson:"time"`
	Content string `json:"content" bson:"content"`
	Source  string `json:"source" bson:"source"`
	Link    string `json:"link" bson:"link"`
}
