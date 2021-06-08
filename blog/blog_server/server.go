package main

import (
	"blog/blogpb"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var (
	collection *mongo.Collection

	articleTitleIdx = "article:title"
)

type server struct {
}

type blogItem struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	AuthorID string             `bson:"author_id"`
	Content  string             `bson:"content"`
	Title    string             `bson:"title"`
	Tags     []string           `bson:"tags"`
}

func (*server) CreateBlog(ctx context.Context, req *blogpb.CreateBlogRequest) (*blogpb.CreateBlogResponse, error) {
	fmt.Println("Create blog request")
	blog := req.GetBlog()

	data := blogItem{
		AuthorID: blog.GetAuthorId(),
		Title:    blog.GetTitle(),
		Content:  blog.GetContent(),
		Tags:     blog.GetTags(),
	}

	res, err := collection.InsertMany(ctx, []interface{}{data})
	//res, err := collection.InsertOne(ctx, data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}
	//oid, ok := res.InsertedID.(primitive.ObjectID)
	oid, ok := res.InsertedIDs[0].(primitive.ObjectID)
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			"Cannot convert to OID",
		)
	}

	return &blogpb.CreateBlogResponse{
		Blog: &blogpb.Blog{
			Id:       oid.Hex(),
			AuthorId: blog.GetAuthorId(),
			Title:    blog.GetTitle(),
			Content:  blog.GetContent(),
			Tags:     blog.GetTags(),
		},
	}, nil

}

func (*server) ReadBlog(ctx context.Context, req *blogpb.ReadBlogRequest) (*blogpb.ReadBlogResponse, error) {
	fmt.Println("Read blog request")

	blogID := req.GetBlogId()
	oid, err := primitive.ObjectIDFromHex(blogID)
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot parse ID",
		)
	}

	// create an empty struct
	data := &blogItem{}
	filter := bson.M{"_id": oid}
	// regex example
	// {"username" : {$regex : ".*son.*"}} // contains "son" in the username

	res := collection.FindOne(ctx, filter)
	if err := res.Decode(data); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, status.Errorf(
				codes.NotFound,
				fmt.Sprintf("Cannot find blog with specified ID: %v", err),
			)
		}
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}

	return &blogpb.ReadBlogResponse{
		Blog: dataToBlogPb(data),
	}, nil
}

func dataToBlogPb(data *blogItem) *blogpb.Blog {
	return &blogpb.Blog{
		Id:       data.ID.Hex(),
		AuthorId: data.AuthorID,
		Content:  data.Content,
		Title:    data.Title,
		Tags:     data.Tags,
	}
}

func (*server) UpdateBlog(ctx context.Context, req *blogpb.UpdateBlogRequest) (*blogpb.UpdateBlogResponse, error) {
	fmt.Println("Update blog request")
	blog := req.GetBlog()
	oid, err := primitive.ObjectIDFromHex(blog.GetId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot parse ID",
		)
	}

	// create an empty struct
	data := &blogItem{}
	filter := bson.M{"_id": oid}

	// we update our internal struct
	data.ID = oid
	data.AuthorID = blog.GetAuthorId()
	data.Content = blog.GetContent()
	data.Title = blog.GetTitle()
	data.Tags = blog.GetTags()

	res, updateErr := collection.UpdateOne(
		ctx,
		filter,
		bson.M{
			"$set": bson.M{
				"author_id": data.AuthorID,
				"content":   data.Content,
				"title":     data.Title,
				"tags":      data.Tags,
			},
		},
		options.Update().SetUpsert(true),
	)
	//res, updateErr := collection.ReplaceOne(ctx, filter, data)
	if updateErr != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot update object in MongoDB: %v", updateErr),
		)
	}

	if res.MatchedCount == 0 {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find blog in MongoDB: %v", updateErr),
		)
	}
	fmt.Printf("matched: %v, modified: %v\n", res.MatchedCount, res.ModifiedCount)

	return &blogpb.UpdateBlogResponse{
		Blog: dataToBlogPb(data),
	}, nil

}

func (*server) DeleteBlog(ctx context.Context, req *blogpb.DeleteBlogRequest) (*blogpb.DeleteBlogResponse, error) {
	fmt.Println("Delete blog request")
	oid, err := primitive.ObjectIDFromHex(req.GetBlogId())
	if err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"Cannot parse ID",
		)
	}

	filter := bson.M{"_id": oid}

	res, err := collection.DeleteOne(ctx, filter)

	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot delete object in MongoDB: %v", err),
		)
	}

	if res.DeletedCount == 0 {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find blog in MongoDB: %v", err),
		)
	}

	return &blogpb.DeleteBlogResponse{BlogId: req.GetBlogId()}, nil
}

func (*server) ListBlog(req *blogpb.ListBlogRequest, stream blogpb.BlogService_ListBlogServer) error {
	fmt.Println("List blog request")

	// D is an ordered representation of a BSON document
	// Example usage: bson.D{{"foo", "bar"}, {"hello", "world"}, {"pi", 3.14159}}
	cur, err := collection.Find(context.Background(), primitive.D{{}})
	if err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}
	defer cur.Close(context.Background())
	for cur.Next(context.Background()) {
		data := &blogItem{}
		err := cur.Decode(data)
		if err != nil {
			return status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error while decoding data from MongoDB: %v", err),
			)

		}
		stream.Send(&blogpb.ListBlogResponse{Blog: dataToBlogPb(data)})
	}
	if err := cur.Err(); err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}
	return nil
}

func (*server) ListBlogPage(ctx context.Context, req *blogpb.ListBlogPageRequest) (*blogpb.ListBlogPageResponse, error) {
	fmt.Println("List blog page request")
	findOptions := options.Find()
	findOptions.SetSkip(req.GetSkip()).SetLimit(req.GetLimit()) // skip and limit default set to 0
	// sorts the documents first by the author_id field in descending order
	// and then by the title field in ascending order
	findOptions.SetSort(bson.M{
		"author_id": -1,
		"title":     1,
	})
	// The maximum number of documents to be included in each batch returned by the server
	// default: 101
	findOptions.SetBatchSize(200)
	// select fields
	findOptions.SetProjection(bson.M{
		"_id":       1,
		"author_id": 1,
		"title":     1,
	})
	/*
		filter := primitive.D{
			{
				"title",
				primitive.D{
					{
						"$in",
						primitive.A{"My Title", "My First Blog (edited)", 3},
					},
				},
			},
		}
	*/
	/*
		filter := bson.M{
			"author_id": bson.M{
				"$not": bson.M{
					"$eq": "Stephane",
				},
			},
			"title": bson.M{
				"$in": primitive.A{"My Title", "My First Blog (edited)", 3},
			},
		}
	*/
	filter := bson.M{
		"$or": []interface{}{
			bson.M{
				"author_id": bson.M{
					"$not": bson.M{
						"$eq": "Stephane",
					},
				},
			},
			bson.M{
				"title": bson.M{
					"$in": primitive.A{"My Title", "My Second Title", 3},
				},
			},
		},
	}
	cur, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}
	defer cur.Close(ctx)

	var resp blogpb.ListBlogPageResponse
	for cur.Next(ctx) {
		data := &blogItem{}
		err := cur.Decode(data)
		if err != nil {
			return nil, status.Errorf(
				codes.Internal,
				fmt.Sprintf("Error while decoding data from MongoDB: %v", err),
			)
		}
		resp.Blogs = append(resp.Blogs, dataToBlogPb(data))
	}
	if err := cur.Err(); err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}
	return &resp, nil
}

func main() {
	// if we crash the go code, we get the file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Println("Connecting to MongoDB")
	// connect to MongoDB
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	err = client.Connect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Blog Service Started")
	collection = client.Database("mydb").Collection("blog")
	indexName, err := collection.Indexes().CreateOne(
		context.Background(),
		mongo.IndexModel{
			// compound index
			Keys: bson.M{
				// descending order
				"article_id": -1,
				// ascending order
				"title": 1,
			},
			// set this index unique
			Options: options.Index().SetUnique(true).SetName(articleTitleIdx),
		},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("create index: ", indexName)
	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{}
	s := grpc.NewServer(opts...)
	blogpb.RegisterBlogServiceServer(s, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)

	go func() {
		fmt.Println("Starting Server...")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	// Wait for Control C to exit
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	// Block until a signal is received
	<-ch
	// First we close the connection with MongoDB:
	fmt.Println("Closing MongoDB Connection")
	// client.Disconnect(context.TODO())
	if err := client.Disconnect(context.TODO()); err != nil {
		log.Fatalf("Error on disconnection with MongoDB : %v", err)
	}
	// Second step : closing the listener
	fmt.Println("Closing the listener")
	if err := lis.Close(); err != nil {
		log.Fatalf("Error on closing the listener : %v", err)
	}
	// Finally, we stop the server
	fmt.Println("Stopping the server")
	// s.Stop()
	s.GracefulStop()
	fmt.Println("End of Program")
}
