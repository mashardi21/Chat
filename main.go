package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"
	"sync"

	_ "github.com/go-sql-driver/mysql"

	chat "github.com/mashardi21/chat/proto"
	"google.golang.org/grpc"
	glog "google.golang.org/grpc/grpclog"
)

var grpcLog glog.LoggerV2

func init() {
	grpcLog = glog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)
}

type Connection struct {
	stream   chat.Broadcast_CreateStreamServer
	userName string
	name     string
	email    string
	active   bool
	error    chan error
}

type Server struct {
	Connection []*Connection
}

func addUserToDatabase(conn *Connection) error {
	db, err := sql.Open("mysql", "mason:H0hew7eu955@tcp(127.0.0.1:3306)/chat")
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
		return err
	}

	defer db.Close()

	insert, err := db.Query("insert into Users (Users_Name, Users_Username, Users_Email) values (?, ?, ?)", conn.name, conn.userName, conn.email)
	if err != nil {
		log.Fatalf("Unable to insert into table: %v", err)
		return err
	}

	defer insert.Close()

	return nil
}

func (s *Server) CreateStream(pconn *chat.Connect, stream chat.Broadcast_CreateStreamServer) error {
	conn := &Connection{
		stream:   stream,
		userName: pconn.User.UserName,
		email:    pconn.User.Email,
		name:     pconn.User.Name,
		active:   true,
		error:    make(chan error),
	}

	addUserToDatabase(conn)

	s.Connection = append(s.Connection, conn)

	return <-conn.error
}

func (s *Server) BroadcastMessage(ctx context.Context, msg *chat.Message) (*chat.Close, error) {
	wait := sync.WaitGroup{}
	done := make(chan int)

	for _, conn := range s.Connection {
		wait.Add(1)

		go func(msg *chat.Message, conn *Connection) {
			defer wait.Done()

			if conn.active {
				err := conn.stream.Send(msg)
				grpcLog.Info("Sending message to: ", conn.userName)

				if err != nil {
					log.Fatalf("Could not send message: %v", err)
				}
			}
		}(msg, conn)
	}

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done
	return &chat.Close{}, nil
}

func main() {
	var connections []*Connection

	server := &Server{connections}

	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error creating the server: %v", err)
	}

	grpcLog.Info("Starting server at port :8080")

	chat.RegisterBroadcastServer(grpcServer, server)
	grpcServer.Serve(listener)
}
