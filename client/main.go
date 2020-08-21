package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	chat "github.com/mashardi21/chat/proto"
	"google.golang.org/grpc"
)

var client chat.BroadcastClient
var wait *sync.WaitGroup

func init() {
	wait = &sync.WaitGroup{}
}

func Connect(user *chat.User) error {
	var streamerror error

	stream, err := client.CreateStream(context.Background(), &chat.Connect{
		User:   user,
		Active: true,
	})

	if err != nil {
		return fmt.Errorf("Connection failed: %v", err)
	}

	wait.Add(1)
	go func(str chat.Broadcast_CreateStreamClient) {
		defer wait.Done()

		for {
			msg, err := str.Recv()
			if err != nil {
				streamerror = fmt.Errorf("Error reading message: %v", err)
				break
			}

			fmt.Printf("%v: %s\n", msg.Name, msg.Body)
		}
	}(stream)

	return streamerror
}

func main() {
	timestamp := time.Now()
	done := make(chan int)

	name := flag.String("n", "Anon", "The name of the user")
	userName := flag.String("u", "Anon", "The username of the user")
	email := flag.String("e", "Example@example.com", "The email of the user")
	flag.Parse()

	/**db, err := sql.Open("mysql", "mason:H0hew7eu955@tcp(127.0.0.1:3306)/chat")

	if err != nil {
		panic(err.Error())
	}

	defer db.Close()

	insert, err := db.Query("insert into Users (Users_Name, Users_Username, Users_Email) values (?, ?, ?)", name, userName, email)

	if err != nil {
		panic(err.Error())
	}

	defer insert.Close()

	rows, err := db.Query("select ID from Users where Users_Username=?", userName)

	if err != nil {
		panic(err.Error())
	}

	var id int64

	for rows.Next() {

		if err := rows.Scan(&id); err != nil {
			panic(err.Error())
		}
	}

	defer rows.Close()**/

	conn, err := grpc.Dial("192.168.0.172:8080", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Couldn't connect to service: %v", err)
	}

	client = chat.NewBroadcastClient(conn)
	user := &chat.User{
		UserName: *userName,
		Name:     *name,
		Email:    *email,
	}

	Connect(user)

	wait.Add(1)

	go func() {
		defer wait.Done()

		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {
			msg := &chat.Message{
				Name:      user.Name,
				Body:      scanner.Text(),
				Timestamp: timestamp.String(),
			}

			_, err := client.BroadcastMessage(context.Background(), msg)

			if err != nil {
				fmt.Printf("Error sending message: %v", err)
				break
			}
		}
	}()

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done

}
