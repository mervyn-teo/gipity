package main

import (
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v2"
)

type optionMap = map[string]*discordgo.ApplicationCommandInteractionDataOption

var (
	allKeys keys
)

// keys struct
type keys struct {
	discord struct {
		Key   string `yaml:"api-key"`
		App   string `yaml:"app"`
		Guild string `yaml:"guild"`
	} `yaml:"discord"`
	chatgpt struct {
		Key string `yaml:"api-key"`
	} `yaml:"chat-gpt"`
}

func processError(err error) {
	log.Fatal(err)
}

func main() {
	// load api keys
	f, err := os.Open("keys.yml")
	if err != nil {
		processError(err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&allKeys)
	if err != nil {
		processError(err)
	}

	// init discord bot
	session, _ := discordgo.New("Bot " + allKeys.discord.Key)

	session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type != discordgo.InteractionApplicationCommand {
			return
		}

		data := i.ApplicationCommandData()
		if data.Name != "gpt" {
			return
		}

		handleGpt(s, i, parseOptions(data.Options))
	})
	session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as %s", r.User.String())
	})

	_, err = session.ApplicationCommandBulkOverwrite(allKeys.discord.App, allKeys.discord.Guild, commands)
	if err != nil {
		log.Fatalf("could not register commands: %s", err)
	}

	err = session.Open()
	if err != nil {
		log.Fatalf("could not open session: %s", err)
	}

	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	err = session.Close()
	if err != nil {
		log.Printf("could not close session gracefully: %s", err)
	}

}

func handleGpt(s *discordgo.Session, i *discordgo.InteractionCreate, opts optionMap) {
	builder := new(strings.Builder)
	if v, ok := opts["author"]; ok && v.BoolValue() {
		author := interactionAuthor(i.Interaction)
		builder.WriteString("**hi! " + author.String() + "**, here is what chat-gpt responded")
	}

	builder.WriteString(opts["message"].StringValue()) //TODO: change this to chat-gpt response

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: builder.String(),
		},
	})

	if err != nil {
		log.Panicf("could not respond to interaction: %s", err)
	}
}

// discord functions
var commands = []*discordgo.ApplicationCommand{
	{
		Name:        "gpt",
		Description: "Ask chat-gpt something",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "message",
				Description: "Contents of the message",
				Type:        discordgo.ApplicationCommandOptionString,
				Required:    true,
			},
		},
	},
}

func parseOptions(options []*discordgo.ApplicationCommandInteractionDataOption) (om optionMap) {
	om = make(optionMap)
	for _, opt := range options {
		om[opt.Name] = opt
	}
	return
}

func interactionAuthor(i *discordgo.Interaction) *discordgo.User {
	if i.Member != nil {
		return i.Member.User
	}
	return i.User
}

// chat-gpt logic
