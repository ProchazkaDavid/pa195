package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v7"
)

var redisClient *redis.Client

func init() {
	db, err := strconv.Atoi(os.Getenv("REDIS_DATABASE"))
	if err != nil {
		log.Fatalln(err)
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	})
}

func fetchAll(limit int) ([]FetchRoom, error) {
	var fRooms []FetchRoom

	start := time.Now()
	messages, err := fetchMessages()
	if err != nil {
		return nil, err
	}

	if len(messages) == 0 {
		fmt.Println("Redis is empty, looking into postgres...")

		start = time.Now()
		messages, err = RetrieveAllMessages(limit)
		if err != nil {
			return nil, err
		}
		t := time.Now()

		fmt.Println("Saving messages from postgres to redis...")
		fmt.Printf("POSTGRES: %d items, %d milliseconds\n", len(messages), t.Sub(start).Milliseconds())

		for _, m := range messages {
			if err := m.save(); err != nil {
				return nil, err
			}
		}
	} else {
		t := time.Now()
		fmt.Printf("REDIS: %d items, %d milliseconds\n", len(messages), t.Sub(start).Milliseconds())
	}

	for _, m := range messages {
		rInFRooms := -1

		// look if room is already in fRooms
		for i, fr := range fRooms {
			if fr.Room == m.Room {
				rInFRooms = i
				break
			}
		}
		if rInFRooms == -1 {
			// room is not in fRooms, add it there
			fRooms = append(fRooms, FetchRoom{
				Room: m.Room,
				Msgs: []Msg{},
			})
			rInFRooms = len(fRooms) - 1
		}

		// add message to the room
		fRooms[rInFRooms].Msgs = append(fRooms[rInFRooms].Msgs, Msg{
			Sender: m.Sender,
			Date:   m.Date,
			Text:   m.Text,
		})
	}

	return fRooms, nil
}

func fetchMessages() ([]Message, error) {
	var messages []Message

	mess, err := redisClient.LRange("messages", 0, -1).Result()
	if err != nil {
		return nil, err
	}

	for _, m := range mess {
		var message Message
		message.UnmarshalBinary([]byte(m))
		messages = append(messages, message)
	}
	return messages, nil
}

func fetchRooms() ([]Room, error) {
	var rooms []Room

	rms, err := redisClient.LRange("rooms", 0, -1).Result()
	if err != nil {
		return nil, err
	}

	for _, r := range rms {
		var room Room
		room.UnmarshalBinary([]byte(r))
		rooms = append(rooms, room)
	}

	return rooms, nil
}

func (m Message) save() error {
	mess, err := m.MarshalBinary()
	if err != nil {
		return err
	}

	if _, err := redisClient.RPush("messages", mess).Result(); err != nil {
		return err
	}

	return nil
}

func (r Room) save() error {
	room, err := r.MarshalBinary()
	if err != nil {
		return err
	}

	if _, err := redisClient.RPush("rooms", room).Result(); err != nil {
		return err
	}

	return nil
}
