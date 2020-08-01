package main

import (
	"github.com/labstack/gommon/log"
	"github.com/sbekti/broadcastd/broadcast"
	"github.com/sbekti/broadcastd/config"
	"github.com/sbekti/broadcastd/server"
	"os"
	"os/signal"
)

func main() {
	c, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	b := broadcast.NewBroadcast(c)
	s := server.NewServer(c.BindIP, c.BindPort, b)
	go func() {
		if err := s.Start(); err != nil {
			log.Info("Server is shutting down.")
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	<-quit

	if err := s.Shutdown(); err != nil {
		log.Fatal(err)
	}
}

//func createLive() {
//cookie := ""
//client, err := instagram.ImportFromString(cookie)
//
//if err != nil {
//	log.Fatalf("Error: %v\n", err)
//}
//log.Printf("Logged IN as: %s\n", client.Account.FullName)

//resp, err := client.Logout()
//if err != nil {
//	log.Fatalf("Error: %v\n", err)
//}
//log.Printf("Logged out: %+v\n", resp)

//live, err := client.Live.Create(720, 1280, "hehe")
//if err != nil {
//	log.Fatalf("Error: %v\n", err)
//}
//log.Printf("LIVE: \n\n%+v\n\n", live)
//
//log.Println(live.UploadURL)
//
//startLive, err := client.Live.Start(live.BroadcastID, false)
//if err != nil {
//	log.Fatalf("Error: %+v\n", err)
//}
//log.Printf("Start: %+v\n", startLive)
//
//unmuteComment, err := client.Live.UnmuteComment(live.BroadcastID)
//if err != nil {
//	log.Fatalf("Error: %+v\n", err)
//}
//log.Printf("unmuteComment: %+v\n", unmuteComment)
//
//killSignal := make(chan os.Signal, 1)
//signal.Notify(killSignal, os.Interrupt)
//go func() {
//	lastCommentTS := 0
//
//	for {
//		log.Printf("==============\n lastCommentTS: %d\n", lastCommentTS)
//		comments, err := client.Live.GetComment(live.BroadcastID, 4, lastCommentTS)
//		if err != nil {
//			log.Fatalf("Error: %v\n", err)
//		}
//		log.Printf("comments: %+v\n", comments)
//
//		for _, comment := range comments.Comments {
//			log.Printf("------------ %s %d: %s", comment.User.Username, comment.CreatedAt, comment.Text)
//
//			if comment.CreatedAt > lastCommentTS {
//				lastCommentTS = comment.CreatedAt
//			}
//		}
//		log.Println("------------")
//
//		heartbeat, err := client.Live.HeartbeatAndGetViewerCount(live.BroadcastID)
//		if err != nil {
//			log.Fatalf("Error: %v\n", err)
//		}
//		log.Printf("heartbeat: %+v\n", heartbeat)
//
//		time.Sleep(5 * time.Second)
//	}
//}()
//<-killSignal
//
//endLive, err := client.Live.End(live.BroadcastID, false)
//if err != nil {
//	log.Fatalf("Error: %+v\n", err)
//}
//log.Printf("End: %v\n", endLive)
//
//addToPost, err := client.Live.AddToPostLive(live.BroadcastID)
//if err != nil {
//	log.Fatalf("Error: %+v\n", err)
//}
//log.Printf("addToPost: %+v\n", addToPost)
//}

//func loginInstagram() {
//	client := instagram.New(
//		os.Getenv("IG_USERNAME"),
//		os.Getenv("IG_PASSWORD"),
//	)
//
//	if err := client.Login(); err != nil {
//		switch v := err.(type) {
//		case instagram.ChallengeError:
//			err := client.Challenge.Process(v.Challenge.APIPath)
//			if err != nil {
//				log.Fatal(err)
//			}
//
//			ui := &input.UI{
//				Writer: os.Stdout,
//				Reader: os.Stdin,
//			}
//
//			query := "What is SMS code for instagram?"
//			code, err := ui.Ask(query, &input.Options{
//				Default:  "000000",
//				Required: true,
//				Loop:     true,
//			})
//			if err != nil {
//				log.Fatal(err)
//			}
//
//			err = client.Challenge.SendSecurityCode(code)
//			if err != nil {
//				log.Fatal(err)
//			}
//
//			client.Account = client.Challenge.LoggedInUser
//		default:
//			log.Fatal(err)
//		}
//	}
//
//	log.Printf("logged in as %s \n", client.Account.FullName)
//
//	result, err := instagram.ExportToString(client)
//	if err != nil {
//		log.Fatal(err)
//	}
//	log.Info(result)
//}
