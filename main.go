package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Todo struct {
	ID primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Text string `json:"text"`
	Completed bool `json:"completed"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

var collection  *mongo.Collection

func main(){

	if os.Getenv("ENV") != "production" {
		err := godotenv.Load(".env"); if err != nil {
			panic(err)
		}
	}
	
	MONGODB_URI := os.Getenv("MONGODB_URI")
	clientOptions := options.Client().ApplyURI(MONGODB_URI)
	client, err := mongo.Connect(context.Background(), clientOptions)

	if err != nil {
		log.Fatal(err)
	}

	defer client.Disconnect(context.Background())

	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MONGODB ATLAS")

	collection = client.Database("golang_db").Collection("todos")

	app := fiber.New()
	app.Use(cors.New(cors.Config{
    AllowOrigins: "http://localhost:5173",  // Explicitly allow React's dev server
    AllowHeaders: "Origin, Content-Type, Accept",
    AllowMethods: "GET,POST,PUT,DELETE,PATCH",
}))

	app.Get("/api/healthcheck", helthCheck)
	app.Get("/api/todos", getTodos)
	app.Post("/api/todos", createTodo)
	app.Patch("/api/todos/:id", updateTodo)
	app.Delete("/api/todos/:id", deleteTodo)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}

	if os.Getenv("ENV") == "production" {
		app.Static("/", "./client/dist")
	}

	log.Fatal(app.Listen("0.0.0.0:" + port))
}

func helthCheck(c *fiber.Ctx) error {
	return c.Status(200).SendString("OK")
}

func getTodos(c *fiber.Ctx) error {
	var todos []*Todo

	cursor, err := collection.Find(context.Background(), bson.M{}); if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	// close the db connection 
	defer cursor.Close(context.Background())


	for cursor.Next(context.Background()) {
		var todo Todo
		cursor.Decode(&todo)
		todos = append(todos, &todo)
	}

	return c.JSON(todos)
}

func createTodo(c *fiber.Ctx) error {
	todo := new(Todo)

	if err := c.BodyParser(todo); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	if todo.Text == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Todo text cannot be empty"})
	}


	result, err := collection.InsertOne(context.Background(), todo); if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Todo text cannot be empty"})
	}

	todo.ID = result.InsertedID.(primitive.ObjectID)

	return c.Status(201).JSON(todo)
}


func updateTodo(c *fiber.Ctx) error {
	id := c.Params("id")
	objectId, err := primitive.ObjectIDFromHex(id); if err != nil {
		return c.Status(400).SendString(err.Error())
	}

	filter := bson.M{"_id": objectId}
	update := bson.M{"$set": bson.M{"completed": true}}

	_, err = collection.UpdateOne(context.Background(), filter, update); if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	return c.Status(200).JSON(fiber.Map{
		"message": "Todo updated",
	})
}



func deleteTodo(c *fiber.Ctx) error {
	id := c.Params("id")
	objectID, err := primitive.ObjectIDFromHex(id); if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid todo ID"})
	}

	filter := bson.M{"_id": objectID}
	_, err = collection.DeleteOne(context.Background(), filter); if err != nil {
		return err
	}
	
	return c.Status(200).JSON(fiber.Map{"success": true})
}