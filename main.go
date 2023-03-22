package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const dbName = "go-blogs-api"
const mongoURI = "mongodb://127.0.0.1:27017/" + dbName 

type MongoInstance struct {
	Db 	   *mongo.Database
	Client *mongo.Client
}

type Blog struct {
	Id          string	`json:"id,omitempty" bson:"_id,omitempty"`
	Title       string 	`json:"title"`
	Body        string	`json:"body"`
	IsPublished bool 	`json:"isPublished"`	
}

var mongoInstance MongoInstance

func connectDB() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	if err != nil {
		return err 
	}

	err = client.Connect(ctx)
	db := client.Database(dbName)

	if err != nil {
		return err 
	}

	mongoInstance = MongoInstance{Db: db, Client: client}

	return nil 
}

func main() {
	if err := connectDB(); err != nil {
		log.Fatal(err)
	}

	app := fiber.New()

	app.Get("/api/blogs", func(c *fiber.Ctx) error {
		collection := mongoInstance.Db.Collection("blogs")
		filter := bson.D{{}}

		cursor, err := collection.Find(c.Context(), filter)

		if err != nil {
			return c.SendStatus(500)
		}

		var blogs []Blog = make([]Blog, 0)

		if err := cursor.All(c.Context(), &blogs); err != nil {
			return c.SendStatus(500)
		}

		return c.Status(200).JSON(blogs)
	})

	app.Get("/api/blogs/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		blogId, err := primitive.ObjectIDFromHex(idParam)
		
		if err != nil {
			return c.SendStatus(500)
		}
		
		collection := mongoInstance.Db.Collection("blogs")
		filter := bson.D{{Key: "_id", Value: blogId}}
		foundedRecord := collection.FindOne(c.Context(), filter)

		blog := &Blog{}
		foundedRecord.Decode(blog)

		if blog.Id == "" {
			return c.SendStatus(404)
		}

		return c.Status(200).JSON(blog)
	})

	app.Post("/api/blogs", func (c *fiber.Ctx) error {
		collection := mongoInstance.Db.Collection("blogs")
		blog := new(Blog)

		if err := c.BodyParser(&blog); err != nil {
			return c.SendStatus(500)
		}

		blog.Id = ""
		blog.IsPublished = false 

		insertionResult, err := collection.InsertOne(c.Context(), &blog)

		if err != nil {
			return c.SendStatus(500)
		}

		filter := bson.D{{Key: "_id", Value: insertionResult.InsertedID}}
		createdRecord := collection.FindOne(c.Context(), filter)

		foundBlog := &Blog{}
		createdRecord.Decode(foundBlog)

		return c.Status(201).JSON(foundBlog)
	})

	
	app.Patch("/api/blogs/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		blogId, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return c.SendStatus(500)
		}

		blog := new(Blog)

		if err := c.BodyParser(&blog); err != nil {
			c.SendStatus(500)
		}

		collection := mongoInstance.Db.Collection("blogs")
		filter := bson.D{{Key: "_id", Value: blogId}}
		update := bson.D{{
			Key: "$set",
			Value: bson.D{
				{Key: "title", Value: blog.Title},
				{Key: "body", Value: blog.Body},
			},
		}}

		err = collection.FindOneAndUpdate(c.Context(), filter, update).Err()

		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(404)
			}

			return c.SendStatus(500)
		}

		blog.Id = idParam
		return c.Status(200).JSON(blog)
	})

	app.Delete("/api/blogs/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		blogId, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return c.SendStatus(500)
		}

		collection := mongoInstance.Db.Collection("blogs")
		filter := bson.D{{Key: "_id", Value: blogId}}
		result, err := collection.DeleteOne(c.Context(), filter)

		if err != nil {
			return c.SendStatus(500)
		}

		if result.DeletedCount < 1 {
			return c.SendStatus(404)
		}

		return c.Status(200).JSON("Blog Deleted!")
	})

	log.Fatal(app.Listen(":8080"))
}
