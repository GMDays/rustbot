package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// RustServer represents the information returned from /info/{id}
type RustServer struct {
	Hostname              string `json:"hostname"`
	IP                    string `json:"ip"`
	Port                  string `json:"port"`
	Map                   string `json:"map"`
	OnlineState           string `json:"online_state"`
	Checked               string `json:"checked"`
	PlayersMax            string `json:"players_max"`
	PlayersCurrent        string `json:"players_cur"`
	PlayersAverage        string `json:"players_avg"`
	PlayersMaxMan         string `json:"players_maxman"`
	PlayersMaxForever     string `json:"players_max_forever"`
	PlayersMaxForeverDate string `json:"players_max_forever_date"`
	Bots                  string `json:"bots"`
	Ratings               string `json:"rating"`
	Entities              string `json:"entities"`
	Version               string `json:"version"`
	Seed                  string `json:"seed"`
	Size                  string `json:"size"`
	Uptime                string `json:"uptime"`
	FPS                   string `json:"fps"`
	FPSAverage            string `json:"fps_avg"`
	URL                   string `json:"url"`
	Image                 string `json:"image"`
	OS                    string `json:"os"`
	Memory                string `json:"mem"`
	Country               string `json:"country"`
	CountryFull           string `json:"country_full"`
	ServerMode            string `json:"server_mode"`
	Wipe                  string `json:"wipe_cycle"`
	Queue                 bool
	QueueLine             int
}

type ApexStats struct {
	Results []struct {
        Aid      string `json:"aid"`
        Name     string `json:"name"`
        Platform string `json:"platform"`
        Avatar   string `json:"avatar"`
        Legend   string `json:"legend"`
        Level    string `json:"level"`
        Kills    string `json:"kills"`
    } `json:"results"`
    Totalresults int `json:"totalresults"`
}

type DiscordMessageEmbed struct {
	*discordgo.MessageEmbed
}

func main() {
	// Lookup token from ENV
	Token := os.Getenv("DISCORD_TOKEN")
	if Token == "" {
		fmt.Println("Unable to find token, please make sure DISCORD_TOKEN is set.")
		return
	}

	// Create a new Discord session using Token from DISCORD_TOKEN
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register handlers
	dg.AddHandler(incomingMessageHandler)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Don't finish main goroutine until some sort of system term is received on the sc channel.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Defer to close the Discord session at the end of the main goroutine.
	defer dg.Close()
}

func incomingMessageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m == nil {
		log.Fatal("Something wrong with your Message bruh.")
		return
	}

	// var message [2]string
	// var playerName string
	var playerName string
	var message []string
	message = strings.Split(m.Content, " ")
	if len(message) > 1 {
		playerName = message[1]
	}

	switch message[0] {
	case "!apex":
		{
			url := "https://apextab.com/api/search.php?platform=pc&search=" + playerName
			res, err := http.Get(url)
			if err != nil {
				fmt.Println("url failed")
				log.Fatal(err)
			}

			info, err := ioutil.ReadAll(res.Body)
			if err != nil {
				fmt.Println("ioutil failed")
				log.Fatal(err)
			}

			res.Body.Close()

			var apexInfo ApexStats			
			json.Unmarshal(info, &apexInfo)
			fmt.Println(apexInfo)		

			embed := NewDiscordMsgEmbed().
			    SetTitle(playerName + " these are your statistics you filthy casual").
			    AddField("Kills: ", apexInfo.Results[0].Kills).
			    AddField("Legend:", apexInfo.Results[0].Legend).
			    AddField("Level:", apexInfo.Results[0].Level).
			    SetDescription("Test Description").
			    SetImage("https://danbooru.donmai.us/data/__lifeline_apex_legends_drawn_by_charles_vaughn__ffe4833e5c193dc7de5a774cd27f4624.jpg").
			    SetThumbnail("https://danbooru.donmai.us/data/__lifeline_apex_legends_drawn_by_charles_vaughn__ffe4833e5c193dc7de5a774cd27f4624.jpg").
			    SetAuthor(discordgo.MessageEmbedAuthor{}.Name).			  
			    SetColor(0x00ff00).MessageEmbed

			fmt.Println(embed)
			s.ChannelMessageSendEmbed(m.ChannelID, embed)
		}
	case "!lowpop":
		{
			// this just checks 50 lowpop
			// @TODO make this lookup the ID first and refactor to work
			res, err := http.Get("https://api.rust-servers.info/info/50")
			if err != nil {
				log.Fatal(err)
			}
			// read body and unmarshal it into the RustServer struct
			info, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Fatal(err)
			}
			res.Body.Close()

			var server RustServer
			json.Unmarshal(info, &server)

			checkQueue(server.PlayersCurrent, server.PlayersMax, &server)

			// Create embedded message with the server information (queue, connection info, etc)
			// based on some conditions, like if a queue exists and if the server is online or not.
			switch server.Queue {
			case true:
				line := strconv.Itoa(server.QueueLine)

				embed := &discordgo.MessageEmbed{
					Author:      &discordgo.MessageEmbedAuthor{},
					Color:       0xff0000, // Red
					Description: "Sucks. There's a queue of " + line + " for " + server.Hostname,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Server Information:",
							Value:  "Wipe: " + server.Wipe + "\nMode: " + server.ServerMode + "\nAverage FPS: " + server.FPSAverage + "\nPlayers Online: " + server.PlayersCurrent,
							Inline: true,
						},
						{
							Name:   "Connection String:",
							Value:  "```client.connect " + server.IP + ":" + server.Port + "```",
							Inline: true,
						},
					},
					Image: &discordgo.MessageEmbedImage{
						URL: server.Image,
					},
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: "https://cdn.discordapp.com/avatars/119249192806776836/cc32c5c3ee602e1fe252f9f595f9010e.jpg?size=2048",
					},
					Timestamp: time.Now().Format(time.RFC3339),
					Title:     server.Hostname,
				}

				s.ChannelMessageSendEmbed(m.ChannelID, embed)

			case false:
				embed := &discordgo.MessageEmbed{
					Author:      &discordgo.MessageEmbedAuthor{},
					Color:       0x00ff00, // Green
					Description: "There's NO queue! Leggo zerg! :100:",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Server Information:",
							Value:  "Wipe: " + server.Wipe + "\nMode: " + server.ServerMode + "\nAverage FPS: " + server.FPSAverage + "\nPlayers Online: " + server.PlayersCurrent,
							Inline: true,
						},
						{
							Name:   "Connection Info:",
							Value:  "```client.connect " + server.IP + ":" + server.Port + "```",
							Inline: true,
						},
					},
					Image: &discordgo.MessageEmbedImage{
						URL: server.Image,
					},
					Thumbnail: &discordgo.MessageEmbedThumbnail{
						URL: "https://cdn.discordapp.com/avatars/119249192806776836/cc32c5c3ee602e1fe252f9f595f9010e.jpg?size=2048",
					},
					Timestamp: time.Now().Format(time.RFC3339),
					Title:     server.Hostname,
				}

				s.ChannelMessageSendEmbed(m.ChannelID, embed)
			}
		}
	}
}

func checkQueue(c string, m string, r *RustServer) {
	currentPlayers, err := strconv.Atoi(c)
	if err != nil {
		log.Fatal(err)
	}
	maxPlayers, err := strconv.Atoi(m)
	if err != nil {
		log.Fatal(err)
	}
	diff := (currentPlayers - maxPlayers)
	// if current users minus max users is not a negative number
	// return the difference and set Queue to true.
	if diff > 0 {
		r.Queue = true
		r.QueueLine = diff
	} else {
		r.Queue = false
	}
}

func NewDiscordMsgEmbed() *DiscordMessageEmbed {
	return &DiscordMessageEmbed{&discordgo.MessageEmbed{}}
}

func (e *DiscordMessageEmbed) SetTitle (title string) *DiscordMessageEmbed {
	e.Title = title
	return e
}

func (e *DiscordMessageEmbed) SetDescription (description string) *DiscordMessageEmbed {
	if len(description) > 2048 {
		description = description[:2048]
	}
	e.Description = description
	return e
}

func (e *DiscordMessageEmbed) AddField(name, value string) *DiscordMessageEmbed {
	if len(value) > 1024 {
		value = value[:1024]
	}

	if len(name) > 1024 {
		name = name[:1024]
	}

	e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
		Name:  name,
		Value: value,
	})

	return e

}

func (e *DiscordMessageEmbed) SetFooter(args ...string) *DiscordMessageEmbed {
	iconURL := ""
	text := ""
	proxyURL := ""

	switch {
	case len(args) > 2:
		proxyURL = args[2]
		fallthrough
	case len(args) > 1:
		iconURL = args[1]
		fallthrough
	case len(args) > 0:
		text = args[0]
	case len(args) == 0:
		return e
	}

	e.Footer = &discordgo.MessageEmbedFooter{
		IconURL:      iconURL,
		Text:         text,
		ProxyIconURL: proxyURL,
	}

	return e
}

func (e *DiscordMessageEmbed) SetImage(args ...string) *DiscordMessageEmbed {
	var URL string
	var proxyURL string

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		URL = args[0]
	}
	if len(args) > 1 {
		proxyURL = args[1]
	}
	e.Image = &discordgo.MessageEmbedImage{
		URL:      URL,
		ProxyURL: proxyURL,
	}
	return e
}

func (e *DiscordMessageEmbed) SetThumbnail(args ...string) *DiscordMessageEmbed {
	var URL string
	var proxyURL string

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		URL = args[0]
	}
	if len(args) > 1 {
		proxyURL = args[1]
	}
	e.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL:      URL,
		ProxyURL: proxyURL,
	}
	return e
}

func (e *DiscordMessageEmbed) SetAuthor(args ...string) *DiscordMessageEmbed {
	var (
		name     string
		iconURL  string
		URL      string
		proxyURL string
	)

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		name = args[0]
	}
	if len(args) > 1 {
		iconURL = args[1]
	}
	if len(args) > 2 {
		URL = args[2]
	}
	if len(args) > 3 {
		proxyURL = args[3]
	}

	e.Author = &discordgo.MessageEmbedAuthor{
		Name:         name,
		IconURL:      iconURL,
		URL:          URL,
		ProxyIconURL: proxyURL,
	}

	return e
}

func (e *DiscordMessageEmbed) SetURL(URL string) *DiscordMessageEmbed {
	e.URL = URL
	return e
}

func (e *DiscordMessageEmbed) SetColor(clr int) *DiscordMessageEmbed {
	e.Color = clr
	return e
}

func (e *DiscordMessageEmbed) InlineAllFields() *DiscordMessageEmbed {
	for _, v := range e.Fields {
		v.Inline = true
	}
	return e
}