package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongodbURI string

func init() {
	// TODO: Use something like cobra for env var parsing.
	mongodbURI = os.Getenv("MONGODB_URI")
	if mongodbURI == "" {
		panic("MONGODB_URI environment variable must be set")
	}
}

func main() {
	client, err := mongo.Connect(
		context.Background(),
		options.Client().ApplyURI(mongodbURI),
	)
	if err != nil {
		panic(err)
	}
	defer client.Disconnect(context.Background())

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	salesHandler := &salesHandler{
		collection: client.Database("sample_supplies").Collection("sales"),
	}
	r.Get("/sales", salesHandler.list)

	err = http.ListenAndServe("localhost:8081", r)
	if err != nil {
		log.Print(err)
	}
}

type sale struct {
	ID       primitive.ObjectID `bson:"_id"`
	SaleDate time.Time          `bson:"saleDate"`
	Items    []saleItem         `bson:"items"`
}

type saleItem struct {
	Name     string               `bson:"name"`
	Tags     []string             `bson:"tags"`
	Price    primitive.Decimal128 `bson:"price"`
	Quantity int32                `bson:"quantity"`
}

type salesHandler struct {
	collection *mongo.Collection
}

func (sh *salesHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cur, err := sh.collection.Find(ctx, bson.D{})
	if err != nil {
		// TODO: Sanitize or encode error?
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)

	i := 0
	for cur.Next(ctx) && i < 10 {
		var sale sale
		err := cur.Decode(&sale)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res, err := json.MarshalIndent(sale, "", "\t")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(res)
		w.Write([]byte("\n"))
		i++
	}
}
