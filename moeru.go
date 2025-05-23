package main

import (
	"flag"
	"log"
	"slices"
	"time"

	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/handler"
)

// Copied from arikawa
const (
	// the limit of max messages per request, as imposed by Discord
	maxMessageFetchLimit = 100
)

func main() {
	token := flag.String("token", "", "Discord Token")
	chunks := flag.Int("chunks", 4, "Distribute message deletion in chunks")
	wait := flag.Int64("wait", 4, "Duration in seconds to wait after completing a chunk")
	dryRun := flag.Bool("dryrun", false, "Operate in dry run mode")
	channel := flag.Int64("channel", 0, "Channel ID")
	flag.Parse()

	h := handler.New()
	s := state.NewAPIOnlyState(*token, h)
	s.Client.AcquireOptions.DontWait = true

	if *channel == 0 {
		log.Fatal("channel id must be given")
	} else if *token == "" {
		log.Fatal("token must be given")
	}

	u, err := s.Me()
	if err != nil {
		log.Fatalln("Failed to get token user:", err)
	}

	before := discord.MessageID(0)
	channelID := discord.ChannelID(*channel)

	for {
		msgs, err := s.MessagesBefore(channelID, before, maxMessageFetchLimit)
		if err != nil {
			log.Fatal(err)
		}

		for chunk := range slices.Chunk(
			slices.DeleteFunc(msgs, func(m discord.Message) bool {
				return m.Author.ID != u.ID
			}), *chunks,
		) {
			for _, msg := range chunk {
				log.Println("Deleting", msg.ID)
				deleting := time.Now()

				if !*dryRun {
					if err := s.DeleteMessage(channelID, msg.ID, ""); err != nil {
						log.Fatalf("Failed to delete message %s: %s", msg.ID, err)
					}
				}

				log.Println("Deleted", msg.ID, "in", time.Since(deleting))
			}

			log.Println("Waiting", *wait, "seconds...")
			time.Sleep(time.Duration(*wait) * time.Second)
		}

		if len(msgs) < maxMessageFetchLimit {
			break
		}

		before = msgs[len(msgs)-1].ID
	}
}
